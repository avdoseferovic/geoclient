//go:build !js

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/avdoseferovic/geoclient/internal/assets"
)

func TestLoadRuntimeConfigManifestOverridesEmbeddedDefaults(t *testing.T) {
	t.Setenv("EO_ASSET_BASE", "")
	t.Setenv("EO_SERVER_ADDR", "")
	t.Setenv("EO_UPDATE_MANIFEST_URL", "")
	t.Setenv("EO_UPDATE_PUBLIC_KEY", "")

	originalServerAddr := defaultServerAddr
	originalAssetBase := defaultAssetBase
	originalUpdateManifestURL := defaultUpdateManifestURL
	originalUpdatePublicKey := defaultUpdatePublicKey
	originalServerConfigKey := desktopServerConfigKey
	t.Cleanup(func() {
		defaultServerAddr = originalServerAddr
		defaultAssetBase = originalAssetBase
		defaultUpdateManifestURL = originalUpdateManifestURL
		defaultUpdatePublicKey = originalUpdatePublicKey
		desktopServerConfigKey = originalServerConfigKey
	})

	server, publicKey := newSignedManifestServer(t, []byte(`{
		"server_addr": "wss://manifest.example/ws",
		"asset_base": "https://manifest.example/assets-v2"
	}`))
	defer server.Close()

	defaultServerAddr = "wss://default.example/ws"
	defaultAssetBase = "https://default.example/assets-v1"
	defaultUpdateManifestURL = server.URL + "/stable.json"
	defaultUpdatePublicKey = publicKey
	desktopServerConfigKey = filepath.Join(t.TempDir(), "server-config.json")

	cfg := loadRuntimeConfig()

	if cfg.serverAddr != "wss://manifest.example/ws" {
		t.Fatalf("server addr = %q, want manifest override", cfg.serverAddr)
	}

	reader, ok := cfg.assetReader.(*assets.VerifiedHTTPReader)
	if !ok {
		t.Fatalf("asset reader type = %T, want *assets.VerifiedHTTPReader", cfg.assetReader)
	}
	if reader.BaseURL != "https://manifest.example/assets-v2" {
		t.Fatalf("asset base = %q, want manifest override", reader.BaseURL)
	}
}

func TestLoadRuntimeConfigEnvOverridesManifest(t *testing.T) {
	t.Setenv("EO_ASSET_BASE", "https://env.example/assets-v3")
	t.Setenv("EO_SERVER_ADDR", "wss://env.example/ws")
	t.Setenv("EO_UPDATE_MANIFEST_URL", "")
	t.Setenv("EO_UPDATE_PUBLIC_KEY", "")

	originalServerAddr := defaultServerAddr
	originalAssetBase := defaultAssetBase
	originalUpdateManifestURL := defaultUpdateManifestURL
	originalUpdatePublicKey := defaultUpdatePublicKey
	originalServerConfigKey := desktopServerConfigKey
	t.Cleanup(func() {
		defaultServerAddr = originalServerAddr
		defaultAssetBase = originalAssetBase
		defaultUpdateManifestURL = originalUpdateManifestURL
		defaultUpdatePublicKey = originalUpdatePublicKey
		desktopServerConfigKey = originalServerConfigKey
	})

	server, publicKey := newSignedManifestServer(t, []byte(`{
		"server_addr": "wss://manifest.example/ws",
		"asset_base": "https://manifest.example/assets-v2"
	}`))
	defer server.Close()

	defaultServerAddr = "wss://default.example/ws"
	defaultAssetBase = "https://default.example/assets-v1"
	defaultUpdateManifestURL = server.URL + "/stable.json"
	defaultUpdatePublicKey = publicKey
	desktopServerConfigKey = filepath.Join(t.TempDir(), "server-config.json")

	cfg := loadRuntimeConfig()

	if cfg.serverAddr != "wss://env.example/ws" {
		t.Fatalf("server addr = %q, want env override", cfg.serverAddr)
	}

	reader, ok := cfg.assetReader.(*assets.VerifiedHTTPReader)
	if !ok {
		t.Fatalf("asset reader type = %T, want *assets.VerifiedHTTPReader", cfg.assetReader)
	}
	if reader.BaseURL != "https://env.example/assets-v3" {
		t.Fatalf("asset base = %q, want env override", reader.BaseURL)
	}
}

func TestUpdateManifestURLPrefersEnv(t *testing.T) {
	t.Setenv("EO_UPDATE_MANIFEST_URL", "https://env.example/channels/stable.json")

	originalUpdateManifestURL := defaultUpdateManifestURL
	t.Cleanup(func() {
		defaultUpdateManifestURL = originalUpdateManifestURL
	})
	defaultUpdateManifestURL = "https://default.example/channels/stable.json"

	if got := updateManifestURL(); got != "https://env.example/channels/stable.json" {
		t.Fatalf("updateManifestURL() = %q, want env override", got)
	}
}

func TestUpdatePublicKeyPrefersEnv(t *testing.T) {
	t.Setenv("EO_UPDATE_PUBLIC_KEY", "env-key")

	originalUpdatePublicKey := defaultUpdatePublicKey
	t.Cleanup(func() {
		defaultUpdatePublicKey = originalUpdatePublicKey
	})
	defaultUpdatePublicKey = "default-key"

	if got := updatePublicKey(); got != "env-key" {
		t.Fatalf("updatePublicKey() = %q, want env override", got)
	}
}

func TestResolveRemoteUpdateManifestReturnsNilForEmptyInputs(t *testing.T) {
	if manifest := resolveRemoteUpdateManifest("", ""); manifest != nil {
		t.Fatalf("resolveRemoteUpdateManifest(\"\", \"\") = %#v, want nil", manifest)
	}
}

func TestLoadRuntimeConfigSavedServerPreferenceOverridesManifest(t *testing.T) {
	t.Setenv("EO_SERVER_ADDR", "")
	t.Setenv("EO_UPDATE_MANIFEST_URL", "")
	t.Setenv("EO_UPDATE_PUBLIC_KEY", "")

	originalServerAddr := defaultServerAddr
	originalUpdateManifestURL := defaultUpdateManifestURL
	originalUpdatePublicKey := defaultUpdatePublicKey
	originalServerConfigKey := desktopServerConfigKey
	t.Cleanup(func() {
		defaultServerAddr = originalServerAddr
		defaultUpdateManifestURL = originalUpdateManifestURL
		defaultUpdatePublicKey = originalUpdatePublicKey
		desktopServerConfigKey = originalServerConfigKey
	})

	server, publicKey := newSignedManifestServer(t, []byte(`{
		"server_addr": "wss://manifest.example/ws",
		"asset_base": "https://manifest.example/assets-v2"
	}`))
	defer server.Close()

	desktopServerConfigKey = filepath.Join(t.TempDir(), "server-config.json")
	if err := saveServerPreference(desktopServerConfigKey, "wss://saved.example/ws"); err != nil {
		t.Fatalf("saveServerPreference() error = %v", err)
	}
	defaultServerAddr = "wss://default.example/ws"
	defaultUpdateManifestURL = server.URL + "/stable.json"
	defaultUpdatePublicKey = publicKey

	cfg := loadRuntimeConfig()
	if cfg.serverAddr != "wss://saved.example/ws" {
		t.Fatalf("server addr = %q, want saved preference override", cfg.serverAddr)
	}
}

func newSignedManifestServer(t *testing.T, payload []byte) (*httptest.Server, string) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stable.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(payload)
		case "/stable.json.sig":
			_, _ = w.Write([]byte(signature))
		default:
			http.NotFound(w, r)
		}
	}))
	return server, base64.StdEncoding.EncodeToString(publicKey)
}
