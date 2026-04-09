//go:build js && wasm

package game

import (
	"log/slog"
	"strings"

	"github.com/avdoseferovic/geoclient/internal/assets"
)

func saveFile(path string, content []byte) error {
	if path == "" {
		return nil
	}
	if !strings.HasPrefix(path, "pub/") {
		slog.Info("skipping non-pub file save on web client", "path", path, "size", len(content))
		return nil
	}
	if err := assets.SaveWebAssetOverride(path, content); err != nil {
		return err
	}
	slog.Info("cached pub file on web client", "path", path, "size", len(content))
	return nil
}
