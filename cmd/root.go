package cmd

import (
	"context"
	"jfrog-assignment/internal/modules/downloader"
	"jfrog-assignment/internal/modules/filereader"
	"jfrog-assignment/internal/modules/persistence"
	"jfrog-assignment/internal/modules/pipeline"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var csvPath string // Path to the CSV file containing URLs, set via command-line flag

var rootCmd = &cobra.Command{
	Use:   "urldownloader",
	Short: "Download content from URLs in a CSV file",
	Long:  `A CLI tool to download content from URLs listed in a CSV file and save them as base64 encoded filenames`,
}

// Execute sets up and runs the root command with the provided context and logger.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - logger: Logger for application-wide logging.
func Execute(ctx context.Context, logger *zap.Logger) {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		run(ctx, logger)
	}
	if err := rootCmd.Execute(); err != nil {
		logger.Error("execution failed", zap.Error(err))
		os.Exit(1)
	}
}

// init initializes the command-line flags for the root command.
func init() {
	rootCmd.Flags().StringVarP(&csvPath, "csv", "c", "", "Path to CSV file containing URLs")
	pflag.CommandLine.AddFlagSet(rootCmd.Flags())
	rootCmd.MarkFlagRequired("csv")
}

// run executes the pipeline to process URLs from the CSV file.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - logger: Logger for logging progress and errors.
func run(ctx context.Context, logger *zap.Logger) {
	p := pipeline.New(logger)
	p.AddStage(filereader.New(csvPath))
	p.AddStage(downloader.New())
	p.AddStage(persistence.New())

	inputChan := make(chan interface{}, 50)
	close(inputChan) // FileReader generates its own input from CSV

	if err := p.Run(ctx, inputChan); err != nil && err != context.Canceled {
		logger.Error("pipeline execution failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("application run completed")
}
