package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type assetFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type assetManifest struct {
	Version     string      `json:"version"`
	BuildCommit string      `json:"build_commit,omitempty"`
	PublishedAt string      `json:"published_at"`
	BaseURL     string      `json:"base_url"`
	Files       []assetFile `json:"files"`
}

type releaseManifest struct {
	Channel           string                     `json:"channel"`
	ClientVersion     string                     `json:"client_version"`
	AssetVersion      string                     `json:"asset_version"`
	ServerAddr        string                     `json:"server_addr"`
	AssetBase         string                     `json:"asset_base"`
	UpdateManifestURL string                     `json:"update_manifest_url,omitempty"`
	Downloads         map[string]releaseDownload `json:"downloads"`
	PublishedAt       string                     `json:"published_at"`
}

type releaseDownload struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

func main() {
	var (
		assetsDir          = flag.String("assets-dir", "", "asset source directory")
		releasesDir        = flag.String("releases-dir", "", "release archives directory")
		assetVersion       = flag.String("asset-version", "", "asset version")
		clientVersion      = flag.String("client-version", "", "client version")
		assetBaseURL       = flag.String("asset-base-url", "", "public base URL for assets")
		releaseBaseURL     = flag.String("release-base-url", "", "public base URL for native archives")
		serverAddr         = flag.String("server-addr", "", "server websocket address")
		buildCommit        = flag.String("build-commit", "", "build commit sha")
		updateManifestURL  = flag.String("update-manifest-url", "", "channel manifest URL")
		assetManifestPath  = flag.String("asset-manifest-out", "", "asset manifest output path")
		releaseManifestOut = flag.String("release-manifest-out", "", "release manifest output path")
		channel            = flag.String("channel", "stable", "release channel")
	)
	flag.Parse()

	now := time.Now().UTC().Format(time.RFC3339)

	if *assetsDir != "" && *assetManifestPath != "" {
		files, err := collectAssetFiles(*assetsDir)
		if err != nil {
			fail(err)
		}
		manifest := assetManifest{
			Version:     *assetVersion,
			BuildCommit: *buildCommit,
			PublishedAt: now,
			BaseURL:     strings.TrimRight(*assetBaseURL, "/"),
			Files:       files,
		}
		if err := writeJSON(*assetManifestPath, manifest); err != nil {
			fail(err)
		}
	}

	if *releaseManifestOut != "" {
		downloads := map[string]releaseDownload{}
		if *releasesDir != "" {
			var err error
			downloads, err = collectDownloads(*releasesDir, strings.TrimRight(*releaseBaseURL, "/"))
			if err != nil {
				fail(err)
			}
		}
		manifest := releaseManifest{
			Channel:           *channel,
			ClientVersion:     *clientVersion,
			AssetVersion:      *assetVersion,
			ServerAddr:        *serverAddr,
			AssetBase:         strings.TrimRight(*assetBaseURL, "/"),
			UpdateManifestURL: *updateManifestURL,
			Downloads:         downloads,
			PublishedAt:       now,
		}
		if err := writeJSON(*releaseManifestOut, manifest); err != nil {
			fail(err)
		}
	}
}

func collectAssetFiles(root string) ([]assetFile, error) {
	var files []assetFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		sum, err := sha256File(path)
		if err != nil {
			return err
		}
		files = append(files, assetFile{
			Path:   filepath.ToSlash(rel),
			Size:   info.Size(),
			SHA256: sum,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func collectDownloads(root, base string) (map[string]releaseDownload, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	downloads := make(map[string]releaseDownload)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "geoclient-") {
			continue
		}
		key := downloadKey(name)
		if key == "" {
			continue
		}
		sum, err := sha256File(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		downloads[key] = releaseDownload{
			URL:    base + "/" + name,
			SHA256: sum,
		}
	}
	return downloads, nil
}

func downloadKey(name string) string {
	trimmed := strings.TrimPrefix(name, "geoclient-")
	trimmed = strings.TrimSuffix(trimmed, ".zip")
	trimmed = strings.TrimSuffix(trimmed, ".tar.gz")
	parts := strings.Split(trimmed, "-")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-2] + "-" + parts[len(parts)-1]
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
