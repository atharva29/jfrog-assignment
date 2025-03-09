package cmd

import (
	"context"
	"jfrog-assignment/internal/modules/downloader"
	"jfrog-assignment/internal/modules/filereader"
	"jfrog-assignment/internal/modules/persistence"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var csvPath string

var rootCmd = &cobra.Command{
	Use:   "urldownloader",
	Short: "Download content from URLs in a CSV file",
	Long:  `A CLI tool to download content from URLs listed in a CSV file and save them as base64 encoded filenames`,
}

// Function variables for dependency injection in tests
var (
	readURLsFunc       = filereader.ReadURLs
	downloadURLsFunc   = downloader.DownloadURLs
	persistContentFunc = persistence.PersistContent
)

func Execute(ctx context.Context, logger *zap.Logger) {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		run(ctx, cmd, args, logger)
	}
	if err := rootCmd.Execute(); err != nil {
		logger.Error("execution failed", zap.Error(err))
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&csvPath, "csv", "c", "", "Path to CSV file containing URLs")
	pflag.CommandLine.AddFlagSet(rootCmd.Flags())
	rootCmd.MarkFlagRequired("csv")
}

func run(ctx context.Context, cmd *cobra.Command, args []string, logger *zap.Logger) {
	urlChan := make(chan string, 50)
	contentChan := make(chan downloader.Content, 50)

	logger.Info("starting URL processing", zap.String("csv_path", csvPath))

	// Use function variables instead of direct calls
	go func() {
		logger.Debug("starting file reader goroutine")
		if err := readURLsFunc(ctx, csvPath, urlChan, logger); err != nil {
			logger.Error("file reading failed", zap.Error(err))
		}
	}()

	go func() {
		logger.Debug("starting downloader goroutine")
		downloadURLsFunc(ctx, urlChan, contentChan, logger)
	}()

	logger.Debug("starting persistence phase")
	if err := persistContentFunc(ctx, contentChan, logger); err != nil {
		logger.Error("persistence failed", zap.Error(err))
	}

	logger.Info("processing completed")
}
