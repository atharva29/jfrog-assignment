package filereader

import (
	"bufio"
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
)

// URLReader defines the interface for reading URLs
type URLReader interface {
	ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error
}

// FileReader implements URLReader
type FileReader struct{}

// New creates a new FileReader
func New() URLReader {
	return &FileReader{}
}

func (fr *FileReader) ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error {
	defer close(urlChan)

	file, err := os.Open(csvPath)
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
				urlChan <- url
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
