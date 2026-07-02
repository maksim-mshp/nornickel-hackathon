package main

import (
	"fmt"
	"os"

	llmgrpc "github.com/maksim-mshp/nornickel-hackathon/internal/llm/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("llm", runtime.WithGRPCService(llmgrpc.NewServer())); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
