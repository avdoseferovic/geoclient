//go:build !js

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func loadInventoryLayout(path string) (map[int]storedInventoryPos, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[int]storedInventoryPos), nil
		}
		return nil, fmt.Errorf("read inventory layout: %w", err)
	}

	var file inventoryLayoutFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode inventory layout: %w", err)
	}
	if file.Positions == nil {
		file.Positions = make(map[int]storedInventoryPos)
	}
	return file.Positions, nil
}

func saveInventoryLayout(path string, positions map[int]storedInventoryPos) error {
	file := inventoryLayoutFile{Positions: positions}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode inventory layout: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write inventory layout: %w", err)
	}
	return nil
}
