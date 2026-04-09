package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/avdoseferovic/geoclient/internal/releaseauth"
)

func resolveRemoteUpdateManifest(url, publicKey string) *updateManifest {
	if url == "" || publicKey == "" {
		return nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	payload, err := releaseauth.FetchSignedBytes(client, url, publicKey)
	if err != nil {
		slog.Debug("update manifest fetch failed", "url", url, "err", err)
		return nil
	}

	var manifest updateManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		slog.Debug("update manifest decode failed", "url", url, "err", err)
		return nil
	}
	return &manifest
}
