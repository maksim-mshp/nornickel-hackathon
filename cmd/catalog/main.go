package main

import (
	"fmt"
	"os"

	cataloggrpc "github.com/maksim-mshp/nornickel-hackathon/internal/catalog/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("catalog", runtime.WithGRPCService(cataloggrpc.NewServer())); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
