package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var (
		filePath   = flag.String("file", "", "file to sign")
		outPath    = flag.String("out", "", "signature output path")
		privateKey = flag.String("private-key", "", "base64 ed25519 private key")
	)
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "--file is required")
		os.Exit(1)
	}
	key := firstNonEmpty(*privateKey, os.Getenv("EO_UPDATE_PRIVATE_KEY"))
	if key == "" {
		fmt.Fprintln(os.Stderr, "private key not provided")
		os.Exit(1)
	}
	if *outPath == "" {
		*outPath = *filePath + ".sig"
	}

	rawKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(rawKey) != ed25519.PrivateKeySize {
		fmt.Fprintln(os.Stderr, "invalid ed25519 private key length")
		os.Exit(1)
	}

	payload, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	signature := ed25519.Sign(ed25519.PrivateKey(rawKey), payload)

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, []byte(base64.StdEncoding.EncodeToString(signature)+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
