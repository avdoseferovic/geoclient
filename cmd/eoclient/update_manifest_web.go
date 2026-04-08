//go:build js && wasm

package main

import (
	"net/url"
	"strings"
	"syscall/js"
)

func updateManifestURL() string {
	search := js.Global().Get("location").Get("search").String()
	values, _ := url.ParseQuery(strings.TrimPrefix(search, "?"))
	return firstNonEmpty(
		values.Get("updateManifestUrl"),
		globalString("__EO_UPDATE_MANIFEST_URL__"),
		defaultUpdateManifestURL,
	)
}

func updatePublicKey() string {
	search := js.Global().Get("location").Get("search").String()
	values, _ := url.ParseQuery(strings.TrimPrefix(search, "?"))
	return firstNonEmpty(
		values.Get("updatePublicKey"),
		globalString("__EO_UPDATE_PUBLIC_KEY__"),
		defaultUpdatePublicKey,
	)
}
