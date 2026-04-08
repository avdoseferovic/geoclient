package assets

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifiedHTTPReaderReadFile(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	asset := []byte("map-data")
	sum := sha256.Sum256(asset)
	manifest := []byte(fmt.Sprintf(`{"files":[{"path":"maps/00001.emf","sha256":"%s"}]}`, hex.EncodeToString(sum[:])))
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, manifest))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/assets/manifest.json":
			_, _ = w.Write(manifest)
		case "/assets/manifest.json.sig":
			_, _ = w.Write([]byte(signature))
		case "/assets/maps/00001.emf":
			_, _ = w.Write(asset)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reader := NewVerifiedHTTPReader(server.URL+"/assets", base64.StdEncoding.EncodeToString(publicKey))
	data, err := reader.ReadFile("maps/00001.emf")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != string(asset) {
		t.Fatalf("ReadFile() = %q, want %q", string(data), string(asset))
	}
}

func TestVerifiedHTTPReaderRejectsTamperedAsset(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	manifest := []byte(`{"files":[{"path":"maps/00001.emf","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, manifest))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/assets/manifest.json":
			_, _ = w.Write(manifest)
		case "/assets/manifest.json.sig":
			_, _ = w.Write([]byte(signature))
		case "/assets/maps/00001.emf":
			_, _ = w.Write([]byte("tampered"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reader := NewVerifiedHTTPReader(server.URL+"/assets", base64.StdEncoding.EncodeToString(publicKey))
	if _, err := reader.ReadFile("maps/00001.emf"); err == nil {
		t.Fatal("ReadFile() error = nil, want checksum failure")
	}
}
