package persistence

import (
	"context"
	"encoding/base64"
	"jfrog-assignment/internal/models"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// FilePersister implements both ContentPersister and pipeline.Stage for saving downloaded content to files.
type FilePersister struct {
	downloadDir string // Directory where files are saved
}

const defaultDownloadDir = "./downloads" // Default directory for saving files

// New creates a new FilePersister instance with an optional custom directory.
//
// Parameters:
//   - downloadDir: Optional variadic parameter for the directory path. Uses defaultDownloadDir if not provided.
//
// Returns:
//   - A pointer to a new FilePersister instance.
func New(downloadDir ...string) *FilePersister {
	dir := defaultDownloadDir
	if len(downloadDir) > 0 {
		dir = downloadDir[0]
	}
	return &FilePersister{downloadDir: dir}
}

// Execute saves content received on the input channel to files as part of the pipeline.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts.
//   - input: Channel to receive content from as interface{}.
//   - output: Output channel (unused, persistence is the final stage).
//   - logger: Logger for logging progress and errors.
//
// Returns:
//   - An error if persistence fails, nil otherwise.
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
