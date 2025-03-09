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

// Content is an alias for models.Content
type Content = models.Content

// URLDownloader defines the interface for downloading URLs (kept for compatibility)
type URLDownloader interface {
	DownloadURLs(ctx context.Context, urlChan <-chan string, contentChan chan<- Content, logger *zap.Logger)
}

// HTTPDownloader implements both URLDownloader and pipeline.Stage
type HTTPDownloader struct{}

const maxWorkers = 50

// New creates a new HTTPDownloader
func New() *HTTPDownloader {
	return &HTTPDownloader{}
}

func (hd *HTTPDownloader) DownloadURLs(ctx context.Context, urlChan <-chan string, contentChan chan<- Content, logger *zap.Logger) {
	urlChanInterface := make(chan interface{}, cap(urlChan))
	contentChanInterface := make(chan interface{}, cap(contentChan))

	// Convert string channel to interface channel
	go func() {
		for url := range urlChan {
			urlChanInterface <- url
		}
		close(urlChanInterface)
	}()

	// Convert interface channel to Content channel
	go func() {
		for content := range contentChanInterface {
			contentChan <- content.(Content)
		}
		close(contentChan)
	}()

	hd.Execute(ctx, urlChanInterface, contentChanInterface, logger)
}

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
				output <- content // Send as interface{}

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
