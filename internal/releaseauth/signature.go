package releaseauth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func ParsePublicKey(encoded string) (ed25519.PublicKey, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, fmt.Errorf("empty public key")
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length %d", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

func VerifyDetached(payload, signature []byte, encodedPublicKey string) error {
	publicKey, err := ParsePublicKey(encodedPublicKey)
	if err != nil {
		return err
	}
	if len(signature) == 0 {
		return fmt.Errorf("empty signature")
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func DecodeSignature(payload []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return nil, fmt.Errorf("empty signature payload")
	}
	signature, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	return signature, nil
}

func FetchSignedBytes(client *http.Client, url, encodedPublicKey string) ([]byte, error) {
	payload, err := fetchBytes(client, url)
	if err != nil {
		return nil, err
	}
	signaturePayload, err := fetchBytes(client, url+".sig")
	if err != nil {
		return nil, err
	}
	signature, err := DecodeSignature(signaturePayload)
	if err != nil {
		return nil, err
	}
	if err := VerifyDetached(payload, signature, encodedPublicKey); err != nil {
		return nil, err
	}
	return payload, nil
}

func fetchBytes(client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: unexpected status %s", url, resp.Status)
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}
	return payload, nil
}
