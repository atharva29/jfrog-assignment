package persistence

import (
	"encoding/base64"
	"fmt"
	"jfrog-assignment/internal/models"
	"os"
	"path/filepath"
)

const downloadDir = "./downloads"

func PersistContent(contentChan <-chan models.Content) error {
	// Create downloads directory
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %v", err)
	}

	// Single goroutine for persistence
	for content := range contentChan {
		if content.Error != nil {
			fmt.Printf("Error processing %s: %v\n", content.URL, content.Error)
			continue
		}

		filename := base64.URLEncoding.EncodeToString([]byte(content.URL)) + ".txt"
		filepath := filepath.Join(downloadDir, filename)

		if err := os.WriteFile(filepath, content.Data, 0644); err != nil {
			fmt.Printf("Failed to save %s: %v\n", content.URL, err)
			continue
		}
		fmt.Printf("Successfully saved %s as %s\n", content.URL, filepath)
	}

	return nil
}
