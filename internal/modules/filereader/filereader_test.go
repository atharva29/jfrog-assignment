package filereader

import (
	"context"
	"os"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestReadURLs(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name         string
		csvContent   string
		expectedURLs []string
		expectErr    bool
	}{
		{
			name:         "valid CSV",
			csvContent:   "Urls\nhttp://example.com\nhttp://test.com\n",
			expectedURLs: []string{"http://example.com", "http://test.com"},
			expectErr:    false,
		},
		{
			name:         "empty CSV",
			csvContent:   "Urls\n",
			expectedURLs: []string{},
			expectErr:    false,
		},
		{
			name:       "invalid file",
			csvContent: "", // Will be tested with non-existent file
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file unless testing invalid file
			var filename string
			if tt.name != "invalid file" {
				tmpFile, err := os.CreateTemp("", "test*.csv")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(tmpFile.Name())
				if _, err := tmpFile.WriteString(tt.csvContent); err != nil {
					t.Fatal(err)
				}
				filename = tmpFile.Name()
				tmpFile.Close()
			} else {
				filename = "nonexistent.csv"
			}

			urlChan := make(chan string, 10)
			done := make(chan error)
			ctx := context.Background()

			go func() {
				done <- ReadURLs(ctx, filename, urlChan, logger)
			}()

			var urls []string
			for url := range urlChan {
				urls = append(urls, url)
			}
			err := <-done

			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(urls) != len(tt.expectedURLs) {
				t.Errorf("expected %d URLs, got %d", len(tt.expectedURLs), len(urls))
			}
			for i, url := range urls {
				if i < len(tt.expectedURLs) && url != tt.expectedURLs[i] {
					t.Errorf("expected URL %s, got %s", tt.expectedURLs[i], url)
				}
			}
		})
	}
}
