# URL Downloader

A command-line tool to download content from URLs listed in a CSV file and save them as files with base64-encoded filenames.

## Installation

### Prerequisites
- Go 1.23.4 or later
- Git (for cloning the repository)


## Usage
```
go mod tidy
go build -o urldownloader .
./urldownloader -c path/to/urls.csv
```

## Critical Design Decision
### Pipeline Module
The pipeline module was added to create a modular, stage-based architecture. It enables extensibility by allowing new stages to be easily plugged into the URL processing flow.


## Unit Tests by Module

### Overall
```
go test ./...
```

### By Module
```
go test ./internal/modules/filereader
go test ./internal/modules/downloader
go test ./internal/modules/persistence
go test ./internal/modules/pipeline
go test ./cmd
```
