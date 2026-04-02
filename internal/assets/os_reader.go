package assets

import "os"

// OSReader reads assets from the local filesystem.
type OSReader struct{}

func NewOSReader() OSReader {
	return OSReader{}
}

func (OSReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
