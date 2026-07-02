package main

import (
	"fmt"
	"os"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/runtime"
)

func main() {
	if err := runtime.Run("llm"); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
