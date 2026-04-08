package main

import "github.com/avdo/eoweb/internal/assets"

type runtimeConfig struct {
	serverAddr      string
	serverConfigKey string
	gfxDir          string
	mapsDir         string
	itemPubPath     string
	npcPubPath      string
	spellPubPath    string
	classPubPath    string
	layoutPath      string
	assetReader     assets.Reader
	windowTitle     string
	defaultWidth    int
	defaultHeight   int
}
