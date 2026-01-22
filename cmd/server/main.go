package main

import (
	"fmt"
	"os"
)

func main() {
	// Initialize application
	container, err := bootstrap()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Bootstrap error: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		if container.Logger != nil {
			_ = container.Logger.Sync()
		}
	}()

	// Run application
	if err := run(container); err != nil {
		container.Logger.Error("Application failed")

		_, _ = fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
