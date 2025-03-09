package filereader

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// TestFileReader_Execute tests the Execute method of FileReader.
func TestFileReader_Execute(t *testing.T) {
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
			csvContent: "",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Starting subtest: %s", tt.name)

			var filename string
			if tt.name != "invalid file" {
				tmpFile, err := os.CreateTemp("", "test*.csv")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				if _, err := tmpFile.WriteString(tt.csvContent); err != nil {
					t.Fatalf("failed to write to temp file: %v", err)
				}
				filename = tmpFile.Name()
				tmpFile.Close()
			} else {
				filename = "nonexistent.csv"
			}

			fr := New(filename)
			inputChan := make(chan interface{}, 1)
			outputChan := make(chan interface{}, 10)
			close(inputChan) // FileReader doesn't need input

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Collect URLs in a separate goroutine
			urls := []string{}
			doneCollecting := make(chan struct{})
			go func() {
				defer close(doneCollecting)
				for url := range outputChan {
					if u, ok := url.(string); ok {
						urls = append(urls, u)
					}
				}
				t.Logf("Finished collecting URLs for %s: %v", tt.name, urls)
			}()

			t.Logf("Executing FileReader for %s", tt.name)
			err := fr.Execute(ctx, inputChan, outputChan, logger)
			close(outputChan) // Close outputChan after Execute returns
			t.Logf("Execute returned for %s with error: %v", tt.name, err)

			// Wait for URL collection to finish or timeout
			select {
			case <-doneCollecting:
				// Check results
				if tt.expectErr && err == nil {
					t.Errorf("expected error, got nil")
				}
				if !tt.expectErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(urls) != len(tt.expectedURLs) {
					t.Errorf("expected %d URLs, got %d: %v", len(tt.expectedURLs), len(urls), urls)
				}
				for i, url := range urls {
					if i < len(tt.expectedURLs) && url != tt.expectedURLs[i] {
						t.Errorf("expected URL %s at index %d, got %s", tt.expectedURLs[i], i, url)
					}
				}
			case <-ctx.Done():
				t.Errorf("test timed out after 5 seconds for %s", tt.name)
			}

			t.Logf("Subtest %s completed", tt.name)
		})
	}
}
