package runtime

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type App struct {
	service      string
	cfg          config.Bundle
	logger       *slog.Logger
	httpServer   *http.Server
	grpcServer   *grpc.Server
	healthServer *health.Server
	grpcServices []GRPCService
	httpServices []HTTPService
	closers      []io.Closer
	workers      []Worker
	readiness    []ReadinessCheck
}

type ReadinessCheck func(context.Context) error

type ReadinessChecker interface {
	ReadinessChecks() []func(context.Context) error
}

type GRPCService interface {
	RegisterGRPC(grpc.ServiceRegistrar)
}

type HTTPService interface {
	RegisterHTTP(chi.Router)
}

type Worker interface {
	Run(ctx context.Context) error
}

type Assembly struct {
	GRPCServices []GRPCService
	HTTPServices []HTTPService
	Closers      []io.Closer
	Workers      []Worker
	Readiness    []func(context.Context) error
}

type HTTPServiceFactory func(config.Bundle, *slog.Logger) (HTTPService, error)

type AssemblyFactory func(config.Bundle, *slog.Logger) (*Assembly, error)

type Option func(*App) error

func WithGRPCService(service GRPCService) Option {
	return func(app *App) error {
		app.grpcServices = append(app.grpcServices, service)
		return nil
	}
}

func WithHTTPService(factory HTTPServiceFactory) Option {
	return func(app *App) error {
		service, err := factory(app.cfg, app.logger)
		if err != nil {
			return err
		}
		app.httpServices = append(app.httpServices, service)
		if closer, ok := service.(io.Closer); ok {
			app.closers = append(app.closers, closer)
		}
		if checker, ok := service.(ReadinessChecker); ok {
			app.addReadiness(checker.ReadinessChecks())
		}
		return nil
	}
}

func WithAssembly(factory AssemblyFactory) Option {
	return func(app *App) error {
		assembly, err := factory(app.cfg, app.logger)
		if err != nil {
			return err
		}
		app.grpcServices = append(app.grpcServices, assembly.GRPCServices...)
		app.httpServices = append(app.httpServices, assembly.HTTPServices...)
		app.closers = append(app.closers, assembly.Closers...)
		app.workers = append(app.workers, assembly.Workers...)
		app.addReadiness(assembly.Readiness)
		return nil
	}
}

func (app *App) addReadiness(checks []func(context.Context) error) {
	for _, check := range checks {
		if check != nil {
			app.readiness = append(app.readiness, check)
		}
	}
}

func Run(service string, options ...Option) error {
	args, err := parseArgs(service)
	if err != nil {
		return err
	}

	cfg, err := config.Load(args.configRoot, args.env, service)
	if err != nil {
		return err
	}

	logger := newLogger(cfg.Runtime.Log)
	app := &App{service: service, cfg: cfg, logger: logger}
	for _, option := range options {
		if err := option(app); err != nil {
			return err
		}
	}

	logger.Info("service starting", "service", service, "env", cfg.Env)

	if err := app.serve(context.Background()); err != nil {
		return err
	}

	logger.Info("service stopped", "service", service)
	return nil
}

type args struct {
	configRoot string
	env        string
}

func parseArgs(service string) (args, error) {
	fs := flag.NewFlagSet(service, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	result := args{}
	fs.StringVar(&result.configRoot, "config", "configs", "configuration root")
	fs.StringVar(&result.env, "env", "dev", "environment")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return args{}, err
	}

	return result, nil
}

func newLogger(cfg config.Log) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	if strings.EqualFold(cfg.Format, "text") {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func (app *App) serve(parent context.Context) error {
	addr := app.cfg.Runtime.Health.Addr
	if addr == "" {
		addr = app.cfg.Runtime.HTTP.Addr
	}
	if addr == "" {
		return errors.New("health listener address is empty")
	}

	router := chi.NewRouter()
	router.Get("/healthz", app.status(http.StatusOK, "ok"))
	router.Get("/readyz", app.readyHandler)
	if app.service == "gateway" {
		router.Get("/", app.gatewayRoot)
	}
	for _, service := range app.httpServices {
		service.RegisterHTTP(router)
	}

	app.httpServer = &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 2)
	go func() {
		app.logger.Info("health listener started", "addr", addr)
		errc <- app.httpServer.ListenAndServe()
	}()

	if app.service != "gateway" {
		if err := app.startGRPC(errc); err != nil {
			_ = app.httpServer.Close()
			return err
		}
	}

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.startWorkers(ctx)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := app.shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown service: %w", err)
		}
		return nil
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve service: %w", err)
	}
}

func (app *App) startGRPC(errc chan<- error) error {
	listener, err := net.Listen("tcp", app.cfg.Runtime.GRPC.Addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	app.grpcServer = grpc.NewServer(app.grpcServerOptions()...)
	app.healthServer = health.NewServer()
	app.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(app.grpcServer, app.healthServer)
	for _, service := range app.grpcServices {
		service.RegisterGRPC(app.grpcServer)
	}
	reflection.Register(app.grpcServer)

	go func() {
		app.logger.Info("grpc listener started", "addr", app.cfg.Runtime.GRPC.Addr)
		errc <- app.grpcServer.Serve(listener)
	}()

	return nil
}

func (app *App) startWorkers(ctx context.Context) {
	for _, worker := range app.workers {
		go func(worker Worker) {
			if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				app.logger.Error("worker stopped", "error", err)
			}
		}(worker)
	}
}

func (app *App) shutdown(ctx context.Context) error {
	if app.healthServer != nil {
		app.healthServer.Shutdown()
	}

	errc := make(chan error, 1)
	go func() {
		errc <- app.httpServer.Shutdown(ctx)
	}()

	if app.grpcServer != nil {
		stopped := make(chan struct{})
		go func() {
			app.grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-ctx.Done():
			app.grpcServer.Stop()
		case <-stopped:
		}
	}

	select {
	case err := <-errc:
		return errors.Join(err, app.closeResources())
	case <-ctx.Done():
		return errors.Join(ctx.Err(), app.closeResources())
	}
}

func (app *App) closeResources() error {
	var result error
	for _, closer := range app.closers {
		if err := closer.Close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func (app *App) status(code int, value string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = fmt.Fprintf(w, `{"service":%q,"status":%q}`+"\n", app.service, value)
	}
}

func (app *App) readyHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	for _, check := range app.readiness {
		if err := check(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"service":%q,"status":"unavailable","error":%q}`+"\n", app.service, err.Error())
			return
		}
	}
	app.status(http.StatusOK, "ready")(w, r)
}

func (app *App) gatewayRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"service":%q,"status":"running"}`+"\n", app.service)
}
