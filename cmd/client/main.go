package main

import (
	"fmt"
	"os"

	"github.com/BigSm0uk/GophKeeper/internal/client/commands"
)

var (
	version   = "dev"
	buildDate = "unknown"
	commit    = "unknown"
)

func main() {
	commands.SetBuildInfo(version, buildDate, commit)

	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "client error: %v\n", err)
		os.Exit(1)
	}
}
