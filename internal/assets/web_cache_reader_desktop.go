//go:build !js

package assets

func NewWebCachedReader(fallback Reader) Reader {
	return fallback
}

func SaveWebAssetOverride(name string, content []byte) error {
	return nil
}
