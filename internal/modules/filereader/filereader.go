package filereader

import (
	"bufio"
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
)

// URLReader defines the interface for reading URLs (kept for compatibility)
type URLReader interface {
	ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error
}

// FileReader implements both URLReader and pipeline.Stage
type FileReader struct {
	csvPath string
}

// New creates a new FileReader
func New(csvPath string) *FileReader {
	return &FileReader{csvPath: csvPath}
}

func (fr *FileReader) ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error {
	urlChanInterface := make(chan interface{}, cap(urlChan))
	go func() {
		for url := range urlChanInterface {
			urlChan <- url.(string)
		}
	}()
	return fr.Execute(ctx, nil, urlChanInterface, logger)
}

func (fr *FileReader) Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error {
	file, err := os.Open(fr.csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	isHeader := true
	urlCount := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			logger.Warn("file reading interrupted", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
			if isHeader {
				isHeader = false
				continue
			}
			url := strings.TrimSpace(scanner.Text())
			if url != "" {
				logger.Debug("read URL", zap.String("url", url))
				output <- url // Send as interface{}
				urlCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	logger.Info("finished reading URLs", zap.Int("total_urls", urlCount))
	return nil
}
