package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestHTTPDownloader_DownloadURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	hd := New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.Write([]byte("test content"))
	}))
	defer ts.Close()

	tsFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tsFail.Close()

	tests := []struct {
		name       string
		url        string
		expectErr  bool
		expectData bool
	}{
		{
			name:       "valid URL",
			url:        ts.URL,
			expectErr:  false,
			expectData: true,
		},
		{
			name:       "invalid URL",
			url:        "http://nonexistent.domain",
			expectErr:  true,
			expectData: false,
		},
		{
			name:       "server error",
			url:        tsFail.URL,
			expectErr:  true,
			expectData: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			contentChan := make(chan Content, 1)
			urlChan := make(chan string, 1)
			urlChan <- tt.url
			close(urlChan)

			go hd.DownloadURLs(ctx, urlChan, contentChan, logger)
			content := <-contentChan

			if tt.expectErr && content.Error == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && content.Error != nil {
				t.Errorf("unexpected error: %v", content.Error)
			}
			if tt.expectData && len(content.Data) == 0 {
				t.Errorf("expected data, got none")
			}
			if !tt.expectData && len(content.Data) > 0 {
				t.Errorf("expected no data, got %s", content.Data)
			}
			if content.Duration <= 0 {
				t.Errorf("expected positive duration, got %d", content.Duration)
			}
		})
	}
}

func TestHTTPDownloader_Cancel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	hd := New()
	urlChan := make(chan string, 1)
	contentChan := make(chan Content, 1)
	ctx, cancel := context.WithCancel(context.Background())

	urlChan <- "http://example.com"
	cancel()

	hd.DownloadURLs(ctx, urlChan, contentChan, logger)
	close(urlChan)
}
