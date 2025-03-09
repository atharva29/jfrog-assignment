package persistence

import (
	"context"
	"encoding/base64"
	"jfrog-assignment/internal/models"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// ContentPersister defines the interface for persisting content
type ContentPersister interface {
	PersistContent(ctx context.Context, contentChan <-chan models.Content, logger *zap.Logger, dir ...string) error
}

// FilePersister implements ContentPersister
type FilePersister struct{}

const defaultDownloadDir = "./downloads"

// New creates a new FilePersister
func New() ContentPersister {
	return &FilePersister{}
}

func (fp *FilePersister) PersistContent(ctx context.Context, contentChan <-chan models.Content, logger *zap.Logger, dir ...string) error {
	downloadDir := defaultDownloadDir
	if len(dir) > 0 {
		downloadDir = dir[0]
	}

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return err
	}

	successCount := 0
	failCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Warn("persistence interrupted", zap.Error(ctx.Err()))
			return ctx.Err()
		case content, ok := <-contentChan:
			if !ok {
				logger.Info("persistence statistics",
					zap.Int("successful", successCount),
					zap.Int("failed", failCount))
				return nil
			}
			if content.Error != nil {
				failCount++
				continue
			}

			filename := base64.URLEncoding.EncodeToString([]byte(content.URL)) + ".txt"
			filepath := filepath.Join(downloadDir, filename)

			logger.Debug("persisting file", zap.String("filepath", filepath))
			if err := os.WriteFile(filepath, content.Data, 0644); err != nil {
				logger.Warn("persist failed",
					zap.String("url", content.URL),
					zap.String("filepath", filepath),
					zap.Error(err))
				failCount++
				continue
			}
			successCount++
		}
	}
}
