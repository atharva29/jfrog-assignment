package downloader

import (
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const downloadDir = "./downloads"

func ProcessURLs(csvPath string) error {
	// Create downloads directory if it doesn't exist
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %v", err)
	}

	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	// Read CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %v", err)
	}

	// Skip header row and process URLs
	for i, record := range records {
		if i == 0 { // Skip header
			continue
		}
		if len(record) == 0 {
			continue
		}

		url := record[0]
		if err := downloadAndSave(url); err != nil {
			fmt.Printf("Failed to process URL %s: %v\n", url, err)
			continue
		}
	}

	return nil
}

func downloadAndSave(url string) error {
	// Ensure URL has protocol
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	// Download content
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code for %s: %d", url, resp.StatusCode)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read content from %s: %v", url, err)
	}

	// Generate filename (base64 of URL)
	filename := base64.URLEncoding.EncodeToString([]byte(url)) + ".txt"
	filepath := filepath.Join(downloadDir, filename)

	// Save to file (will replace if exists)
	if err := os.WriteFile(filepath, content, 0644); err != nil {
		return fmt.Errorf("failed to save content for %s: %v", url, err)
	}

	fmt.Printf("Successfully downloaded and saved %s as %s\n", url, filepath)
	return nil
}
