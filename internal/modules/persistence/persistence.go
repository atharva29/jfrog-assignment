package persistence

import (
	"context"
	"encoding/base64"
	"jfrog-assignment/internal/models"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// ContentPersister defines the interface for persisting content (kept for compatibility)
type ContentPersister interface {
	PersistContent(ctx context.Context, contentChan <-chan models.Content, logger *zap.Logger, dir ...string) error
}

// FilePersister implements both ContentPersister and pipeline.Stage
type FilePersister struct {
	downloadDir string
}

const defaultDownloadDir = "./downloads"

// New creates a new FilePersister
func New(downloadDir ...string) *FilePersister {
	dir := defaultDownloadDir
	if len(downloadDir) > 0 {
		dir = downloadDir[0]
	}
	return &FilePersister{downloadDir: dir}
}

func (fp *FilePersister) PersistContent(ctx context.Context, contentChan <-chan models.Content, logger *zap.Logger, dir ...string) error {
	contentChanInterface := make(chan interface{}, cap(contentChan))
	go func() {
		for content := range contentChan {
			contentChanInterface <- content
		}
		close(contentChanInterface)
	}()
	return fp.Execute(ctx, contentChanInterface, nil, logger)
}

func (fp *FilePersister) Execute(ctx context.Context, input <-chan interface{}, output chan<- interface{}, logger *zap.Logger) error {
	if err := os.MkdirAll(fp.downloadDir, 0755); err != nil {
		return err
	}

	successCount := 0
	failCount := 0

	for content := range input {
		select {
		case <-ctx.Done():
			logger.Warn("persistence interrupted", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
			c, ok := content.(models.Content)
			if !ok {
				logger.Warn("invalid input type, expected Content", zap.Any("type", content))
				continue
			}
			if c.Error != nil {
				failCount++
				continue
			}

			filename := base64.URLEncoding.EncodeToString([]byte(c.URL)) + ".txt"
			filepath := filepath.Join(fp.downloadDir, filename)

			logger.Debug("persisting file", zap.String("filepath", filepath))
			if err := os.WriteFile(filepath, c.Data, 0644); err != nil {
				logger.Warn("persist failed",
					zap.String("url", c.URL),
					zap.String("filepath", filepath),
					zap.Error(err))
				failCount++
				continue
			}
			successCount++
		}
	}

	logger.Info("persistence statistics",
		zap.Int("successful", successCount),
		zap.Int("failed", failCount))
	return nil
}
