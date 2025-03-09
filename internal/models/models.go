package models

type URLRecord struct {
	URL string
}

type Content struct {
	URL      string
	Data     []byte
	Error    error
	Duration int64 // milliseconds
}
