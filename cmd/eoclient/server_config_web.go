//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

func loadServerPreference(path string) (string, error) {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return "", nil
	}
	raw := storage.Call("getItem", path)
	if raw.IsNull() || raw.IsUndefined() || raw.String() == "" {
		return "", nil
	}
	var pref serverPreference
	if err := json.Unmarshal([]byte(raw.String()), &pref); err != nil {
		return "", fmt.Errorf("decode server preference: %w", err)
	}
	return pref.Address, nil
}

func saveServerPreference(path, address string) error {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return nil
	}
	raw, err := json.Marshal(serverPreference{Address: address})
	if err != nil {
		return fmt.Errorf("encode server preference: %w", err)
	}
	storage.Call("setItem", path, string(raw))
	return nil
}
