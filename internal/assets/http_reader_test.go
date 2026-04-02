package assets

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHTTPReaderReadFile(t *testing.T) {
	reader := NewHTTPReader("https://example.test/assets/")
	reader.Client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://example.test/assets/maps/00001.emf" {
				t.Fatalf("unexpected URL %q", req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("map-data")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	data, err := reader.ReadFile("maps/00001.emf")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "map-data" {
		t.Fatalf("ReadFile = %q, want map-data", string(data))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
