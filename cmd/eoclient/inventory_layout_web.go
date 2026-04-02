//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

func loadInventoryLayout(path string) (map[int]storedInventoryPos, error) {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return make(map[int]storedInventoryPos), nil
	}

	raw := storage.Call("getItem", path)
	if raw.IsNull() || raw.IsUndefined() || raw.String() == "" {
		return make(map[int]storedInventoryPos), nil
	}

	var file inventoryLayoutFile
	if err := json.Unmarshal([]byte(raw.String()), &file); err != nil {
		return nil, fmt.Errorf("decode inventory layout: %w", err)
	}
	if file.Positions == nil {
		file.Positions = make(map[int]storedInventoryPos)
	}
	return file.Positions, nil
}

func saveInventoryLayout(path string, positions map[int]storedInventoryPos) error {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return nil
	}

	file := inventoryLayoutFile{Positions: positions}
	raw, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode inventory layout: %w", err)
	}
	storage.Call("setItem", path, string(raw))
	return nil
}
