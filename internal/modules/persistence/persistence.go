package persistence

import (
	"context"
	"encoding/base64"
	"jfrog-assignment/internal/models"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

const downloadDir = "./downloads"

func PersistContent(ctx context.Context, contentChan <-chan models.Content, logger *zap.Logger) error {
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
			if !ok { // Channel closed
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
				logger.Error("persist failed",
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
