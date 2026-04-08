package main

type updateDownload struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type updateManifest struct {
	Channel           string                    `json:"channel"`
	ClientVersion     string                    `json:"client_version"`
	AssetVersion      string                    `json:"asset_version"`
	ServerAddr        string                    `json:"server_addr"`
	AssetBase         string                    `json:"asset_base"`
	UpdateManifestURL string                    `json:"update_manifest_url,omitempty"`
	Downloads         map[string]updateDownload `json:"downloads,omitempty"`
	PublishedAt       string                    `json:"published_at,omitempty"`
}
