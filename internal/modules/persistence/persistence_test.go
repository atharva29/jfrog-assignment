package persistence

import (
	"context"
	"fmt"
	"jfrog-assignment/internal/models"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestFilePersister_Execute(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		contents    []models.Content
		expectErr   bool
		expectFiles int
	}{
		{
			name: "successful persistence",
			contents: []models.Content{
				{URL: "http://example.com", Data: []byte("test data")},
			},
			expectErr:   false,
			expectFiles: 1,
		},
		{
			name: "with error content",
			contents: []models.Content{
				{URL: "http://example.com", Error: fmt.Errorf("download failed")},
				{URL: "http://test.com", Data: []byte("test data")},
			},
			expectErr:   false,
			expectFiles: 1,
		},
		{
			name:        "empty content",
			contents:    []models.Content{},
			expectErr:   false,
			expectFiles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fp := New(tmpDir)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			inputChan := make(chan interface{}, len(tt.contents))
			for _, content := range tt.contents {
				inputChan <- content
			}
			close(inputChan)

			outputChan := make(chan interface{}, 1) // Not used, but required by interface
			done := make(chan error)
			go func() {
				done <- fp.Execute(ctx, inputChan, outputChan, logger)
			}()

			err := <-done

			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			files, _ := filepath.Glob(filepath.Join(tmpDir, "*.txt"))
			if len(files) != tt.expectFiles {
				t.Errorf("expected %d files, got %d", tt.expectFiles, len(files))
			}
		})
	}
}
