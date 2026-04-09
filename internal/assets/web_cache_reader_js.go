//go:build js && wasm

package assets

import (
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"syscall/js"
)

const webAssetCachePrefix = "asset-cache:"

type WebCachedReader struct {
	Fallback Reader
}

func NewWebCachedReader(fallback Reader) Reader {
	if fallback == nil {
		return nil
	}
	return &WebCachedReader{Fallback: fallback}
}

func (r *WebCachedReader) ReadFile(name string) ([]byte, error) {
	clean := strings.TrimLeft(path.Clean(name), "/")
	if data, ok, err := loadWebAssetOverride(clean); err != nil {
		return nil, err
	} else if ok {
		return data, nil
	}
	return r.Fallback.ReadFile(clean)
}

func SaveWebAssetOverride(name string, content []byte) error {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return nil
	}
	clean := strings.TrimLeft(path.Clean(name), "/")
	payload := base64.StdEncoding.EncodeToString(content)
	storage.Call("setItem", webAssetCachePrefix+clean, payload)
	return nil
}

func loadWebAssetOverride(name string) ([]byte, bool, error) {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return nil, false, nil
	}
	raw := storage.Call("getItem", webAssetCachePrefix+name)
	if raw.IsNull() || raw.IsUndefined() || raw.String() == "" {
		return nil, false, nil
	}
	data, err := base64.StdEncoding.DecodeString(raw.String())
	if err != nil {
		return nil, false, fmt.Errorf("decode cached asset %s: %w", name, err)
	}
	return data, true, nil
}
