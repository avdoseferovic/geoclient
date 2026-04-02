//go:build js && wasm

package main

import (
	"net/url"
	"strings"
	"syscall/js"

	"github.com/avdo/eoweb/internal/assets"
)

func loadRuntimeConfig() runtimeConfig {
	search := js.Global().Get("location").Get("search").String()
	values, _ := url.ParseQuery(strings.TrimPrefix(search, "?"))

	assetBase := firstNonEmpty(
		values.Get("assetBase"),
		globalString("__EO_ASSET_BASE__"),
		"/assets",
	)
	serverAddr := firstNonEmpty(
		values.Get("serverAddr"),
		globalString("__EO_SERVER_ADDR__"),
		inferWebSocketAddr(),
	)

	return runtimeConfig{
		serverAddr:    serverAddr,
		gfxDir:        "gfx",
		mapsDir:       "maps",
		itemPubPath:   "pub/dat001.eif",
		npcPubPath:    "pub/dtn001.enf",
		layoutPath:    "inventory-layout.json",
		assetReader:   assets.NewHTTPReader(assetBase),
		windowTitle:   "EO Client Web",
		defaultWidth:  640,
		defaultHeight: 480,
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
