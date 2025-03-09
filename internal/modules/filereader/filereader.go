package filereader

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ReadURLs(csvPath string, urlChan chan<- string) {
	defer close(urlChan)

	file, err := os.Open(csvPath)
	if err != nil {
		urlChan <- fmt.Sprintf("error:%v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	isHeader := true
	for scanner.Scan() {
		if isHeader { // Skip header
			isHeader = false
			continue
		}
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urlChan <- url
		}
	}

	if err := scanner.Err(); err != nil {
		urlChan <- fmt.Sprintf("error:%v", err)
	}
}
