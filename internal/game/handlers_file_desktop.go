//go:build !js

package game

import (
	"log/slog"
	"os"
	"path/filepath"
)

func saveFile(path string, content []byte) error {
	if path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	slog.Info("saving file", "path", path, "size", len(content))
	return os.WriteFile(path, content, 0o644)
}
