package cmd

import (
	"fmt"
	"jfrog-assignment/internal/modules/downloader"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var csvPath string

var rootCmd = &cobra.Command{
	Use:   "urldownloader",
	Short: "Download content from URLs in a CSV file",
	Long:  `A CLI tool to download content from URLs listed in a CSV file and save them as base64 encoded filenames`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := downloader.ProcessURLs(csvPath); err != nil {
			fmt.Printf("Error processing URLs: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&csvPath, "csv", "c", "", "Path to CSV file containing URLs")
	pflag.CommandLine.AddFlagSet(rootCmd.Flags())
	rootCmd.MarkFlagRequired("csv")
}
