package runtime

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func (app *App) grpcServerOptions() []grpc.ServerOption {
	signingKey := []byte(app.cfg.Runtime.Auth.SigningKey)
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			app.requestIDUnaryInterceptor,
			auth.UnaryServerInterceptor(signingKey),
			app.loggingUnaryInterceptor,
			grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(app.recoverPanic)),
		),
		grpc.ChainStreamInterceptor(
			app.requestIDStreamInterceptor,
			auth.StreamServerInterceptor(signingKey),
			app.loggingStreamInterceptor,
			grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandler(app.recoverPanic)),
		),
	}
}

func (app *App) requestIDUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	next := contextWithRequestID(ctx)
	return handler(next, req)
}

func (app *App) requestIDStreamInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return handler(srv, &contextServerStream{
		ServerStream: stream,
		ctx:          contextWithRequestID(stream.Context()),
	})
}

func (app *App) loggingUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	app.logGRPC(ctx, info.FullMethod, err, time.Since(start))
	return resp, err
}

func (app *App) loggingStreamInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, stream)
	app.logGRPC(stream.Context(), info.FullMethod, err, time.Since(start))
	return err
}

func (app *App) recoverPanic(value any) error {
	app.logger.Error("grpc panic recovered", "panic", value, "stack", string(debug.Stack()))
	return status.Error(codes.Internal, "internal server error")
}

func (app *App) logGRPC(ctx context.Context, method string, err error, elapsed time.Duration) {
	code := status.Code(err)
	attrs := []any{
		"service", app.service,
		"method", method,
		"code", code.String(),
		"duration_ms", elapsed.Milliseconds(),
		"request_id", requestIDFromContext(ctx),
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
		app.logger.Warn("grpc request completed", attrs...)
		return
	}
	app.logger.Info("grpc request completed", attrs...)
}

func contextWithRequestID(ctx context.Context) context.Context {
	requestID := requestIDFromMetadata(ctx)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

func requestIDFromMetadata(ctx context.Context) string {
	incoming, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := incoming.Get("x-request-id")
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func requestIDFromContext(ctx context.Context) string {
	value, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}
	return value
}

type contextServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (stream *contextServerStream) Context() context.Context {
	return stream.ctx
}
