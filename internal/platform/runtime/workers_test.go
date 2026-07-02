package runtime

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"google.golang.org/grpc"
)

type fakeWorker struct {
	started chan struct{}
}

func (worker *fakeWorker) Run(ctx context.Context) error {
	close(worker.started)
	<-ctx.Done()
	return ctx.Err()
}

type fakeGRPCService struct{}

func (fakeGRPCService) RegisterGRPC(grpc.ServiceRegistrar) {}

type fakeCloser struct{}

func (fakeCloser) Close() error { return nil }

func TestStartWorkersPropagatesContextCancel(t *testing.T) {
	t.Parallel()

	app := &App{logger: slog.Default()}
	worker := &fakeWorker{started: make(chan struct{})}
	app.workers = []Worker{worker}

	ctx, cancel := context.WithCancel(context.Background())
	app.startWorkers(ctx)

	select {
	case <-worker.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	cancel()
}

func TestWithAssemblyRegistersParts(t *testing.T) {
	t.Parallel()

	app := &App{logger: slog.Default()}
	factory := func(config.Bundle, *slog.Logger) (*Assembly, error) {
		return &Assembly{
			GRPCServices: []GRPCService{fakeGRPCService{}},
			Closers:      []io.Closer{fakeCloser{}},
			Workers:      []Worker{&fakeWorker{started: make(chan struct{})}},
		}, nil
	}
	if err := WithAssembly(factory)(app); err != nil {
		t.Fatalf("WithAssembly: %v", err)
	}
	if len(app.grpcServices) != 1 || len(app.closers) != 1 || len(app.workers) != 1 {
		t.Fatalf("assembly not registered: grpc=%d closers=%d workers=%d",
			len(app.grpcServices), len(app.closers), len(app.workers))
	}
}

func TestWithAssemblyPropagatesFactoryError(t *testing.T) {
	t.Parallel()

	app := &App{logger: slog.Default()}
	factory := func(config.Bundle, *slog.Logger) (*Assembly, error) {
		return nil, errFactory
	}
	if err := WithAssembly(factory)(app); err != errFactory {
		t.Fatalf("expected factory error, got %v", err)
	}
	if len(app.workers) != 0 {
		t.Fatal("no workers should be registered on factory error")
	}
}

var errFactory = errors.New("factory failed")
