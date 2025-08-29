package cmd

import (
	"fmt"
	"runtime"
)

// These variables are populated by the Go linker (`ldflags`) at build time.
var (
	Version   = "dev" // Default value for development builds
	Commit    = "none"
	BuildDate = "unknown"
)

// VersionCmd holds the command for displaying version information.
type VersionCmd struct{}

// Run executes the version command, printing build-time information.
func (v *VersionCmd) Run() error {
	fmt.Printf("WHAM! Version: %s\n", Version)
	fmt.Printf("Commit: %s\n", Commit)
	fmt.Printf("Build Date: %s\n", BuildDate)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	return nil
}
