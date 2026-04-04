package assets

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

// HTTPReader loads assets over HTTP. This is used by the browser client.
type HTTPReader struct {
	BaseURL string
	Client  *http.Client
}

func NewHTTPReader(baseURL string) *HTTPReader {
	return &HTTPReader{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  http.DefaultClient,
	}
}

func (r *HTTPReader) ReadFile(name string) (data []byte, err error) {
	client := r.Client
	if client == nil {
		client = http.DefaultClient
	}

	target := r.BaseURL + "/" + strings.TrimLeft(path.Clean(name), "/")
	resp, err := client.Get(target)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", target, err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close %s: %w", target, closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %s", target, resp.Status)
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", target, err)
	}
	return data, nil
}
