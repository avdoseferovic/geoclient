//go:build !js

package main

import "os"

func updateManifestURL() string {
	if value := os.Getenv("EO_UPDATE_MANIFEST_URL"); value != "" {
		return value
	}
	return defaultUpdateManifestURL
}

func updatePublicKey() string {
	if value := os.Getenv("EO_UPDATE_PUBLIC_KEY"); value != "" {
		return value
	}
	return defaultUpdatePublicKey
}
