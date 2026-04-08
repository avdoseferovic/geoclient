package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/avdo/eoweb/internal/releaseauth"
)

type verifiedAssetManifest struct {
	Files []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"files"`
}

type VerifiedHTTPReader struct {
	BaseURL   string
	PublicKey string
	Client    *http.Client

	manifestOnce sync.Once
	manifest     map[string]string
	manifestErr  error
}

func NewVerifiedHTTPReader(baseURL, publicKey string) Reader {
	if strings.TrimSpace(publicKey) == "" {
		return NewHTTPReader(baseURL)
	}
	return &VerifiedHTTPReader{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		PublicKey: publicKey,
		Client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *VerifiedHTTPReader) ReadFile(name string) ([]byte, error) {
	if err := r.loadManifest(); err != nil {
		return nil, err
	}
	clean := strings.TrimLeft(path.Clean(name), "/")
	expected, ok := r.manifest[clean]
	if !ok {
		return nil, fmt.Errorf("asset %s missing from manifest", clean)
	}
	httpReader := &HTTPReader{BaseURL: r.BaseURL, Client: r.Client}
	data, err := httpReader.ReadFile(clean)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != strings.ToLower(expected) {
		return nil, fmt.Errorf("asset %s failed checksum verification", clean)
	}
	return data, nil
}

func (r *VerifiedHTTPReader) loadManifest() error {
	r.manifestOnce.Do(func() {
		client := r.Client
		if client == nil {
			client = http.DefaultClient
		}
		payload, err := releaseauth.FetchSignedBytes(client, r.BaseURL+"/manifest.json", r.PublicKey)
		if err != nil {
			r.manifestErr = fmt.Errorf("fetch signed asset manifest: %w", err)
			return
		}
		var manifest verifiedAssetManifest
		if err := json.Unmarshal(payload, &manifest); err != nil {
			r.manifestErr = fmt.Errorf("decode asset manifest: %w", err)
			return
		}
		r.manifest = make(map[string]string, len(manifest.Files))
		for _, file := range manifest.Files {
			r.manifest[strings.TrimLeft(path.Clean(file.Path), "/")] = file.SHA256
		}
	})
	return r.manifestErr
}
