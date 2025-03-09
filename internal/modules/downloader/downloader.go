package downloader

import (
	"fmt"
	"io"
	"jfrog-assignment/internal/models"
	"net/http"
	"strings"
	"sync"
)

const maxWorkers = 50

func DownloadURLs(urlChan <-chan string, contentChan chan<- models.Content) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	defer close(contentChan)

	for url := range urlChan {
		if strings.HasPrefix(url, "error:") {
			contentChan <- models.Content{Error: fmt.Errorf(strings.TrimPrefix(url, "error:"))}
			return
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(url string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			content := downloadURL(url)
			contentChan <- content
		}(url)
	}

	wg.Wait()
}

func downloadURL(url string) models.Content {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	resp, err := http.Get(url)
	if err != nil {
		return models.Content{URL: url, Error: fmt.Errorf("download failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Content{URL: url, Error: fmt.Errorf("bad status: %d", resp.StatusCode)}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Content{URL: url, Error: fmt.Errorf("read failed: %v", err)}
	}

	return models.Content{URL: url, Data: data}
}
