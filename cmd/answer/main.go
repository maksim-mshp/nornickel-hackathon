package main

import (
	"fmt"
	"os"

	answergrpc "github.com/maksim-mshp/nornickel-hackathon/internal/answer/ports/grpc"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("answer", runtime.WithGRPCService(answergrpc.NewServer())); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
