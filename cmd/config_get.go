package cmd

import (
	"os"
)

// ConfigCmd represents the 'config' command group.
type ConfigCmd struct {
	Get GetConfigCmd `cmd:"" help:"Show the final, merged configuration."`
}

// GetConfigCmd handles the 'config get' command.
type GetConfigCmd struct{}

// Run executes the 'config get' command, printing the merged configuration.
func (c *GetConfigCmd) Run(ctx *Context) error {
	// This command is designed for structured output. If the user requests 'table'
	// format (which is the CLI default), we'll default to YAML as it's the
	// source format and more human-readable for this kind of data.
	outputFormat := ctx.OutputFormat
	if outputFormat == "table" {
		outputFormat = "yaml"
	}

	// Use the shared helper to render the data, ensuring consistent output handling.
	return RenderData(os.Stdout, ctx.WHAM.Config(), outputFormat)
}
