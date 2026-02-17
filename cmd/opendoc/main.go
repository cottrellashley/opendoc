// OpenDoc CLI — a static site generator with integrated workbench and AI chat.
package main

import (
	"fmt"
	"os"
)

// Version is set at build time by GoReleaser via ldflags.
// Falls back to "dev" for local development builds.
var Version = "dev"

func main() {
	rootCmd.Version = Version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
