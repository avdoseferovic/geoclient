//go:build !js

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func loadServerPreference(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read server preference: %w", err)
	}
	var pref serverPreference
	if err := json.Unmarshal(raw, &pref); err != nil {
		return "", fmt.Errorf("decode server preference: %w", err)
	}
	return pref.Address, nil
}

func saveServerPreference(path, address string) error {
	raw, err := json.MarshalIndent(serverPreference{Address: address}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode server preference: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write server preference: %w", err)
	}
	return nil
}
