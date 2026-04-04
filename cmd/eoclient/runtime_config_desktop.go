//go:build !js

package main

import (
	"os"

	"github.com/avdo/eoweb/internal/assets"
)

func loadRuntimeConfig() runtimeConfig {
	serverAddr := os.Getenv("EO_SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "ws://127.0.0.1:8078"
	}

	return runtimeConfig{
		serverAddr:    serverAddr,
		gfxDir:        "gfx",
		mapsDir:       "maps",
		itemPubPath:   "pub/dat001.eif",
		npcPubPath:    "pub/dtn001.enf",
		layoutPath:    "inventory-layout.json",
		assetReader:   assets.NewOSReader(),
		windowTitle:   "EO Client",
		defaultWidth:  960,
		defaultHeight: 640,
	}
}
