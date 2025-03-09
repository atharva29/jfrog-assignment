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

var csvPath string

var rootCmd = &cobra.Command{
	Use:   "urldownloader",
	Short: "Download content from URLs in a CSV file",
	Long:  `A CLI tool to download content from URLs listed in a CSV file and save them as base64 encoded filenames`,
}

func Execute(ctx context.Context, logger *zap.Logger) {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		run(ctx, logger)
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

func run(ctx context.Context, logger *zap.Logger) {
	p := pipeline.New(logger)
	p.AddStage(filereader.New(csvPath))
	p.AddStage(downloader.New())
	p.AddStage(persistence.New())

	inputChan := make(chan interface{}, 50)
	close(inputChan) // FileReader doesn't need input, starts with CSV

	if err := p.Run(ctx, inputChan); err != nil && err != context.Canceled {
		logger.Error("pipeline execution failed", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("application run completed")
}
