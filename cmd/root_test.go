package cmd

import (
	"context"
	"jfrog-assignment/internal/modules/downloader"
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// Mock implementations
type mockURLReader struct{}

func (m *mockURLReader) ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error {
	urlChan <- "http://example.com"
	close(urlChan)
	return nil
}

type mockURLDownloader struct{}

func (m *mockURLDownloader) DownloadURLs(ctx context.Context, urlChan <-chan string, contentChan chan<- downloader.Content, logger *zap.Logger) {
	for url := range urlChan {
		contentChan <- downloader.Content{URL: url, Data: []byte("test")}
	}
	close(contentChan)
}

type mockContentPersister struct{}

func (m *mockContentPersister) PersistContent(ctx context.Context, contentChan <-chan downloader.Content, logger *zap.Logger, dir ...string) error {
	for range contentChan {
	}
	return nil
}

func TestPipeline(t *testing.T) {
	logger := zaptest.NewLogger(t)

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

	ctx := context.Background()
	csvPath = tmpFile.Name()

	run(ctx, nil, nil, logger,
		&mockURLReader{}, // Use mock directly, no New() needed for mocks
		&mockURLDownloader{},
		&mockContentPersister{},
	)
}
