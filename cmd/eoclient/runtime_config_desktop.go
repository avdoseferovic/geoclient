//go:build !js

package main

import (
	"os"

	"github.com/avdoseferovic/geoclient/internal/assets"
)

var desktopServerConfigKey = "server-config.json"

func loadRuntimeConfig() runtimeConfig {
	assetBase := defaultAssetBase
	serverAddr := defaultServerAddr
	serverConfigKey := desktopServerConfigKey

	if manifest := resolveRemoteUpdateManifest(updateManifestURL(), updatePublicKey()); manifest != nil {
		if manifest.AssetBase != "" {
			assetBase = manifest.AssetBase
		}
		if manifest.ServerAddr != "" {
			serverAddr = manifest.ServerAddr
		}
	}
	if saved, err := loadServerPreference(serverConfigKey); err == nil && saved != "" {
		serverAddr = saved
	}

	if value := os.Getenv("EO_ASSET_BASE"); value != "" {
		assetBase = value
	}
	if value := os.Getenv("EO_SERVER_ADDR"); value != "" {
		serverAddr = value
	}

	if serverAddr == "" {
		serverAddr = "ws://127.0.0.1:8078"
	}

	var assetReader assets.Reader = assets.NewOSReader()
	if assetBase != "" {
		assetReader = assets.NewVerifiedHTTPReader(assetBase, updatePublicKey())
	}

	return runtimeConfig{
		serverAddr:      serverAddr,
		serverConfigKey: serverConfigKey,
		gfxDir:          "gfx",
		mapsDir:         "maps",
		itemPubPath:     "pub/dat001.eif",
		npcPubPath:      "pub/dtn001.enf",
		spellPubPath:    "pub/dsl001.esf",
		classPubPath:    "pub/dat001.ecf",
		layoutPath:      "inventory-layout.json",
		assetReader:     assetReader,
		windowTitle:     "EO Client",
		defaultWidth:    960,
		defaultHeight:   640,
	}
}
