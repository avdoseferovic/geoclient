//go:build !js

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	selfupdateapply "github.com/creativeprojects/go-selfupdate/update"
)

func maybeApplyDesktopBinaryUpdate() bool {
	if os.Getenv("EO_DISABLE_AUTO_UPDATE") == "1" {
		return false
	}
	url := updateManifestURL()
	publicKey := updatePublicKey()
	if url == "" || publicKey == "" || clientVersion == "" || clientVersion == "dev" {
		return false
	}
	if !isSemverVersion(clientVersion) {
		slog.Debug("auto-update disabled for non-semver build", "client_version", clientVersion)
		return false
	}

	manifest := resolveRemoteUpdateManifest(url, publicKey)
	if manifest == nil {
		return false
	}
	if !shouldApplyBinaryUpdate(clientVersion, manifest.ClientVersion) {
		return false
	}

	download, ok := manifest.Downloads[currentPlatformKey()]
	if !ok || download.URL == "" || download.SHA256 == "" {
		slog.Warn("update available but no compatible download found", "platform", currentPlatformKey(), "version", manifest.ClientVersion)
		return false
	}

	targetPath, err := os.Executable()
	if err != nil {
		slog.Warn("auto-update skipped: executable path unavailable", "err", err)
		return false
	}
	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		slog.Warn("auto-update skipped: executable path invalid", "err", err)
		return false
	}

	client := &http.Client{Timeout: 60 * time.Second}
	archiveData, err := downloadAndVerifyArchive(client, download)
	if err != nil {
		slog.Warn("auto-update skipped: download verification failed", "err", err)
		return false
	}
	binaryData, err := extractArchiveBinary(archiveData, filepath.Ext(targetPath) == ".exe")
	if err != nil {
		slog.Warn("auto-update skipped: archive extraction failed", "err", err)
		return false
	}
	if err := applyBinaryUpdate(targetPath, binaryData); err != nil {
		slog.Warn("auto-update skipped: apply failed", "err", err)
		return false
	}
	if err := relaunchUpdatedBinary(targetPath, os.Args[1:]); err != nil {
		slog.Warn("auto-update skipped: relaunch failed", "err", err)
		return false
	}

	slog.Info("applying client update", "from", clientVersion, "to", manifest.ClientVersion)
	return true
}

func downloadAndVerifyArchive(client *http.Client, download updateDownload) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, download.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download archive: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download archive: unexpected status %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download archive: %w", err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != strings.ToLower(download.SHA256) {
		return nil, fmt.Errorf("download archive checksum mismatch")
	}
	return data, nil
}

func extractArchiveBinary(data []byte, wantExe bool) ([]byte, error) {
	if binary, err := extractZipBinary(data, wantExe); err == nil {
		return binary, nil
	}
	return extractTarGzBinary(data, wantExe)
}

func extractZipBinary(data []byte, wantExe bool) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, file := range reader.File {
		name := filepath.Base(file.Name)
		if !isExpectedBinaryName(name, wantExe) {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = rc.Close()
		}()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("binary not found in zip archive")
}

func extractTarGzBinary(data []byte, wantExe bool) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = gzipReader.Close()
	}()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if !isExpectedBinaryName(filepath.Base(header.Name), wantExe) {
			continue
		}
		return io.ReadAll(tarReader)
	}
	return nil, fmt.Errorf("binary not found in tar.gz archive")
}

func applyBinaryUpdate(targetPath string, binaryData []byte) error {
	return selfupdateapply.Apply(bytes.NewReader(binaryData), selfupdateapply.Options{
		TargetPath: targetPath,
		TargetMode: 0o755,
	})
}

func relaunchUpdatedBinary(targetPath string, args []string) error {
	cmd := exec.Command(targetPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func currentPlatformKey() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

func shouldApplyBinaryUpdate(currentVersion, targetVersion string) bool {
	if targetVersion == "" || currentVersion == "" || currentVersion == targetVersion {
		return false
	}
	currentParts, currentOK := parseVersionTriplet(currentVersion)
	targetParts, targetOK := parseVersionTriplet(targetVersion)
	if currentOK && targetOK {
		return compareVersionTriplets(currentParts, targetParts) < 0
	}
	return false
}

func isSemverVersion(version string) bool {
	_, ok := parseVersionTriplet(version)
	return ok
}

func parseVersionTriplet(version string) ([3]int, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(version, "v"))
	parts := strings.SplitN(trimmed, "-", 2)
	fields := strings.Split(parts[0], ".")
	if len(fields) == 0 || len(fields) > 3 {
		return [3]int{}, false
	}
	var result [3]int
	for i, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil {
			return [3]int{}, false
		}
		result[i] = value
	}
	return result, true
}

func compareVersionTriplets(a, b [3]int) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func isExpectedBinaryName(name string, wantExe bool) bool {
	if wantExe {
		return strings.EqualFold(name, "geoclient.exe")
	}
	return name == "geoclient"
}
