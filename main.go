package main

import (
	"log"
	"os"
	"time"

	"github.com/rs/zerolog"
	cmd "matiq.ai/wham/cmd"
)

func main() {
	var cli cmd.CLI

	ctxKong := cmd.Parse(&cli)

	// The 'version' command does not need configuration or a WHAM instance.
	// We handle it here as a special case to avoid the mandatory config loading.
	if ctxKong.Command() == "version" {
		err := ctxKong.Run()
		ctxKong.FatalIfErrorf(err)
		return
	}

	// Initialize Zerolog.
	var logger zerolog.Logger
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	// Disable color output if NO_COLOR environment variable is set. This is useful for testing.
	if os.Getenv("NO_COLOR") != "" {
		output.NoColor = true
	}

	// Create a logger instance with a level based on the --debug flag.
	logLevel := zerolog.InfoLevel
	if cli.Debug {
		logLevel = zerolog.DebugLevel
	}
	logger = zerolog.New(output).Level(logLevel).With().Timestamp().Logger()

	// Set zerolog as the default logger for messages printed with log.Printf.
	log.SetFlags(0)
	log.SetOutput(logger)

	// Load WHAM configuration.
	config, err := cmd.LoadConfig(cli.Config...)
	if err != nil {
		logger.Fatal().Err(err).Strs("config_paths", cli.Config).Msg("Failed to load WHAM configuration.")
	}

	// Create the WHAM instance.
	wham, err := cmd.NewWHAM(config, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize WHAM engine.")
	}

	// Create the data and metadata directories if they do not exist.
	// This is done after the WHAM instance is created because NewWHAM resolves
	// the directory paths to be absolute, ensuring they are created in the correct location.
	if err := os.MkdirAll(wham.Config().WhamSettings.MetadataDir, 0755); err != nil {
		logger.Fatal().Err(err).Str("dir", wham.Config().WhamSettings.MetadataDir).Msg("Failed to create metadata directory.")
	}
	if err := os.MkdirAll(wham.Config().WhamSettings.DataDir, 0755); err != nil {
		logger.Fatal().Err(err).Str("dir", wham.Config().WhamSettings.DataDir).Msg("Failed to create data directory.")
	}

	// Create the context to be passed to the CLI command handlers.
	cmdCtx := &cmd.Context{
		WHAM:         wham,
		Logger:       logger,
		OutputFormat: cli.Output, // Pass the global output format to the context.
	}

	// Run the selected command.
	err = ctxKong.Run(cmdCtx)
	if err != nil {
		logger.Fatal().Err(err).Msg("WHAM command failed.")
	}
}
