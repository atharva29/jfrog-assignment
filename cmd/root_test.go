package cmd

import (
	"context"
	"jfrog-assignment/internal/modules/downloader"
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestPipeline(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp CSV file
	tmpFile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString("Urls\nhttp://example.com\n")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Save original functions
	origReadURLs := readURLsFunc
	origDownloadURLs := downloadURLsFunc
	origPersistContent := persistContentFunc

	// Restore originals after test
	defer func() {
		readURLsFunc = origReadURLs
		downloadURLsFunc = origDownloadURLs
		persistContentFunc = origPersistContent
	}()

	// Mock implementations
	readURLsFunc = func(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error {
		urlChan <- "http://example.com"
		close(urlChan)
		return nil
	}
	downloadURLsFunc = func(ctx context.Context, urlChan <-chan string, contentChan chan<- downloader.Content, logger *zap.Logger) {
		for url := range urlChan {
			contentChan <- downloader.Content{URL: url, Data: []byte("test")}
		}
		close(contentChan)
	}
	persistContentFunc = func(ctx context.Context, contentChan <-chan downloader.Content, logger *zap.Logger, dir ...string) error {
		for range contentChan {
		}
		return nil
	}

	ctx := context.Background()
	csvPath = tmpFile.Name()
	run(ctx, nil, nil, logger)

	// If we reach here without panicking or blocking, pipeline worked
}
