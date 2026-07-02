package main

import (
	"fmt"
	"os"

	epistemicgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/epistemic/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("epistemic", runtime.WithGRPCService(epistemicgrpc.NewServer())); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
