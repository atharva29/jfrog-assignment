package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestHTTPDownloader_Execute(t *testing.T) {
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			inputChan := make(chan interface{}, 1)
			outputChan := make(chan interface{}, 1)
			inputChan <- tt.url
			close(inputChan)

			done := make(chan error)
			go func() {
				done <- hd.Execute(ctx, inputChan, outputChan, logger)
			}()

			content := <-outputChan
			err := <-done
			if err != nil {
				t.Errorf("Execute failed unexpectedly: %v", err)
			}

			c, ok := content.(Content)
			if !ok {
				t.Fatalf("expected Content type, got %T", content)
			}

			if tt.expectErr && c.Error == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && c.Error != nil {
				t.Errorf("unexpected error: %v", c.Error)
			}
			if tt.expectData && len(c.Data) == 0 {
				t.Errorf("expected data, got none")
			}
			if !tt.expectData && len(c.Data) > 0 {
				t.Errorf("expected no data, got %s", c.Data)
			}
			if c.Duration <= 0 {
				t.Errorf("expected positive duration, got %d", c.Duration)
			}
		})
	}
}

func TestHTTPDownloader_Cancel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	hd := New()

	ctx, cancel := context.WithCancel(context.Background())
	inputChan := make(chan interface{}, 1)
	outputChan := make(chan interface{}, 1)

	inputChan <- "http://example.com"
	cancel()

	done := make(chan error)
	go func() {
		done <- hd.Execute(ctx, inputChan, outputChan, logger)
	}()

	err := <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	close(inputChan)
}
