package downloader

import (
	"context"
	"fmt"
	"io"
	"jfrog-assignment/internal/models"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Content is an alias for models.Content, representing downloaded content.
type Content = models.Content

// HTTPDownloader implements both URLDownloader and pipeline.Stage for downloading content from URLs.
type HTTPDownloader struct{}

const maxWorkers = 50 // Maximum number of concurrent download workers

// New creates a new HTTPDownloader instance.
//
// Returns:
//   - A pointer to a new HTTPDownloader instance.
func New() *HTTPDownloader {
	return &HTTPDownloader{}
}

// Execute downloads content from URLs received on the input channel and sends results to the output channel.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - input: Channel to receive URLs from as interface{}.
//   - output: Channel to send downloaded content to as interface{}.
//   - logger: Logger for logging progress and errors.
//
// Returns:
//   - An error if execution fails (currently always nil unless context is canceled).
func (hd *HTTPDownloader) Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error {
	var (
		wg           sync.WaitGroup
		semaphore    = make(chan struct{}, maxWorkers)
		successCount int32
		failCount    int32
		totalDur     int64
	)

	for url := range input {
		urlStr, ok := url.(string)
		if !ok {
			logger.Warn("invalid input type, expected string", zap.Any("type", url))
			continue
		}
		select {
		case <-ctx.Done():
			logger.Warn("download interrupted", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
			wg.Add(1)
			semaphore <- struct{}{}

			go func(url string) {
				defer wg.Done()
				defer func() { <-semaphore }()

				logger.Debug("downloading URL", zap.String("url", url))
				content := downloadURL(ctx, url)
				output <- content

				if content.Error != nil {
					logger.Warn("download failed",
						zap.String("url", url),
						zap.Error(content.Error))
					atomic.AddInt32(&failCount, 1)
				} else {
					logger.Debug("download successful", zap.String("url", url))
					atomic.AddInt32(&successCount, 1)
					atomic.AddInt64(&totalDur, content.Duration)
				}
			}(urlStr)
		}
	}

	wg.Wait()

	var avgDur float64
	if successCount > 0 {
		avgDur = float64(totalDur) / float64(successCount)
	}

	logger.Info("download statistics",
		zap.Int32("successful", successCount),
		zap.Int32("failed", failCount),
		zap.Float64("avg_duration_ms", avgDur))
	return nil
}

// downloadURL performs the HTTP request to download content from a single URL.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - url: The URL to download.
//
// Returns:
//   - A Content struct with the result (data or error) and duration.
func downloadURL(ctx context.Context, url string) Content {
	start := time.Now()

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Content{
			URL:      url,
			Error:    fmt.Errorf("request creation failed: %v", err),
			Duration: time.Since(start).Milliseconds(),
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Content{
			URL:      url,
			Error:    fmt.Errorf("download failed: %v", err),
			Duration: time.Since(start).Milliseconds(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Content{
			URL:      url,
			Error:    fmt.Errorf("bad status: %d", resp.StatusCode),
			Duration: time.Since(start).Milliseconds(),
		}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Content{
			URL:      url,
			Error:    fmt.Errorf("read failed: %v", err),
			Duration: time.Since(start).Milliseconds(),
		}
	}

	duration := time.Since(start).Milliseconds()
	if duration == 0 {
		duration = 1
	}

	return Content{
		URL:      url,
		Data:     data,
		Duration: duration,
	}
}
