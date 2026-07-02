package main

import (
	"fmt"
	"log/slog"
	"os"

	gatewayhttp "github.com/maksim-mshp/nornickel-hackathon/internal/gateway/ports/http"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("gateway", runtime.WithHTTPService(newHTTPServer)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newHTTPServer(cfg config.Bundle, logger *slog.Logger) (runtime.HTTPService, error) {
	return gatewayhttp.NewServer(cfg, logger)
}
