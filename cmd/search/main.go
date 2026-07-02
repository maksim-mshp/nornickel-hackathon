package main

import (
	"fmt"
	"os"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
	searchgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/search/ports/grpc"
)

func main() {
	if err := runtime.Run("search", runtime.WithGRPCService(searchgrpc.NewServer())); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
