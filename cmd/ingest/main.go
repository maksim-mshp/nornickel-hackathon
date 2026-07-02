package main

import (
	"fmt"
	"os"

	ingestgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/ingest/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("ingest", runtime.WithGRPCService(ingestgrpc.NewServer())); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
