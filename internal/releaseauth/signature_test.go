package releaseauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyDetached(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	payload := []byte("hello")
	signature := ed25519.Sign(privateKey, payload)

	err = VerifyDetached(payload, signature, base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatalf("VerifyDetached() error = %v", err)
	}
}

func TestFetchSignedBytes(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	payload := []byte(`{"ok":true}`)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stable.json":
			_, _ = w.Write(payload)
		case "/stable.json.sig":
			_, _ = w.Write([]byte(signature))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	got, err := FetchSignedBytes(server.Client(), server.URL+"/stable.json", base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatalf("FetchSignedBytes() error = %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload = %q, want %q", string(got), string(payload))
	}
}

func TestFetchSignedBytesRejectsBadSignature(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stable.json":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/stable.json.sig":
			_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString([]byte("bad-signature"))))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if _, err := FetchSignedBytes(server.Client(), server.URL+"/stable.json", base64.StdEncoding.EncodeToString(publicKey)); err == nil {
		t.Fatal("FetchSignedBytes() error = nil, want verification failure")
	}
}
