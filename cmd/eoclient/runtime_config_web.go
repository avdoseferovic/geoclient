//go:build js && wasm

package main

import (
	"net/url"
	"strings"
	"syscall/js"

	"github.com/avdoseferovic/geoclient/internal/assets"
)

func loadRuntimeConfig() runtimeConfig {
	search := js.Global().Get("location").Get("search").String()
	values, _ := url.ParseQuery(strings.TrimPrefix(search, "?"))
	serverConfigKey := "server-config"

	assetBase := firstNonEmpty(
		globalString("__EO_ASSET_BASE__"),
	)
	serverAddr := firstNonEmpty(
		globalString("__EO_SERVER_ADDR__"),
		inferWebSocketAddr(),
	)
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
	assetBase = firstNonEmpty(
		values.Get("assetBase"),
		assetBase,
	)
	serverAddr = firstNonEmpty(
		values.Get("serverAddr"),
		serverAddr,
	)

	return runtimeConfig{
		serverAddr:      serverAddr,
		serverConfigKey: serverConfigKey,
		gfxDir:          "gfx",
		mapsDir:         "maps",
		itemPubPath:     "pub/dat001.eif",
		npcPubPath:      "pub/dtn001.enf",
		layoutPath:      "inventory-layout.json",
		assetReader:     assets.NewWebCachedReader(assets.NewVerifiedHTTPReader(assetBase, updatePublicKey())),
		windowTitle:     "EO Client Web",
		defaultWidth:    640,
		defaultHeight:   480,
	}
}

func inferWebSocketAddr() string {
	location := js.Global().Get("location")
	protocol := "ws://"
	if location.Get("protocol").String() == "https:" {
		protocol = "wss://"
	}
	host := location.Get("hostname").String()
	return protocol + host + ":8078"
}

func globalString(name string) string {
	value := js.Global().Get(name)
	if value.IsUndefined() || value.IsNull() {
		return ""
	}
	return value.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
