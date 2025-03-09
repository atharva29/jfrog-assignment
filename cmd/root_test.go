package cmd

import (
	"context"
	"jfrog-assignment/internal/modules/downloader"
	"jfrog-assignment/internal/modules/pipeline"
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type mockURLReader struct{}

func (m *mockURLReader) ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error {
	urlChan <- "http://example.com"
	close(urlChan)
	return nil
}
func (m *mockURLReader) Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error {
	output <- "http://example.com"
	return nil
}

type mockURLDownloader struct{}

func (m *mockURLDownloader) DownloadURLs(ctx context.Context, urlChan <-chan string, contentChan chan<- downloader.Content, logger *zap.Logger) {
	for url := range urlChan {
		contentChan <- downloader.Content{URL: url, Data: []byte("test")}
	}
	close(contentChan)
}
func (m *mockURLDownloader) Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error {
	for url := range input {
		if u, ok := url.(string); ok {
			output <- downloader.Content{URL: u, Data: []byte("test")}
		}
	}
	return nil
}

type mockContentPersister struct{}

func (m *mockContentPersister) PersistContent(ctx context.Context, contentChan <-chan downloader.Content, logger *zap.Logger, dir ...string) error {
	for range contentChan {
	}
	return nil
}
func (m *mockContentPersister) Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error {
	for range input {
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

	p := pipeline.New(logger)
	csvPath = tmpFile.Name()
	p.AddStage(&mockURLReader{})
	p.AddStage(&mockURLDownloader{})
	p.AddStage(&mockContentPersister{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan interface{}, 50)
	close(inputChan)

	done := make(chan error)
	go func() {
		done <- p.Run(ctx, inputChan)
	}()

	err = <-done
	if err != nil {
		t.Errorf("pipeline execution failed: %v", err)
	}
}
