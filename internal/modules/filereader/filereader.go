package filereader

import (
	"bufio"
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
)

// URLReader defines the interface for reading URLs from a source.
// This is kept for backward compatibility with non-pipeline usage.
type URLReader interface {
	ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error
}

// FileReader implements both URLReader and pipeline.Stage for reading URLs from a CSV file.
type FileReader struct {
	csvPath string // Path to the CSV file containing URLs
}

// New creates a new FileReader instance with the specified CSV file path.
//
// Parameters:
//   - csvPath: The path to the CSV file to read URLs from.
//
// Returns:
//   - A pointer to a new FileReader instance.
func New(csvPath string) *FileReader {
	return &FileReader{csvPath: csvPath}
}

// ReadURLs reads URLs from the CSV file and sends them to the provided channel.
// This method adapts the pipeline Execute method for compatibility with the URLReader interface.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - csvPath: Path to the CSV file (ignored, uses instance csvPath).
//   - urlChan: Channel to send URLs to.
//   - logger: Logger for logging progress and errors.
//
// Returns:
//   - An error if reading fails, nil otherwise.
func (fr *FileReader) ReadURLs(ctx context.Context, csvPath string, urlChan chan<- string, logger *zap.Logger) error {
	urlChanInterface := make(chan interface{}, cap(urlChan))
	go func() {
		defer close(urlChanInterface)
		for url := range urlChanInterface {
			urlChan <- url.(string)
		}
	}()
	return fr.Execute(ctx, nil, urlChanInterface, logger)
}

// Execute reads URLs from the CSV file and sends them to the output channel as part of the pipeline.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - input: Input channel (unused, FileReader generates its own data).
//   - output: Channel to send URLs to as interface{}.
//   - logger: Logger for logging progress and errors.
//
// Returns:
//   - An error if reading fails, nil otherwise.
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
