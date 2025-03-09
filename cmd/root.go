package cmd

import (
	"fmt"
	"jfrog-assignment/internal/models"
	"jfrog-assignment/internal/modules/downloader"
	"jfrog-assignment/internal/modules/filereader"
	"jfrog-assignment/internal/modules/persistence"
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
		urlChan := make(chan string, 50)             // Buffer for URLs
		contentChan := make(chan models.Content, 50) // Buffer for downloaded content

		// Start file reader (Stage 1)
		go filereader.ReadURLs(csvPath, urlChan)

		// Start downloader (Stage 2)
		go downloader.DownloadURLs(urlChan, contentChan)

		// Start persistence (Stage 3)
		if err := persistence.PersistContent(contentChan); err != nil {
			fmt.Printf("Error persisting content: %v\n", err)
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
