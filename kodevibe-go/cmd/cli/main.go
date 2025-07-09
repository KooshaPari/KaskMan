package main

import (
	"fmt"
	"os"

	"github.com/kooshapari/kodevibe-go/internal/cli"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
	commit    = "unknown"
)

func main() {
	// Create root command
	rootCmd := cli.NewRootCommand(version, buildTime, commit)

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}