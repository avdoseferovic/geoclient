package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	var (
		filePath     = flag.String("file", "", "file to sign")
		outPath      = flag.String("out", "", "signature output path")
		privateKey   = flag.String("private-key", "", "base64 ed25519 private key")
		keychainSvc  = flag.String("keychain-service", firstNonEmpty(os.Getenv("EO_UPDATE_PRIVATE_KEY_KEYCHAIN_SERVICE"), "geoclient-update-private-key"), "macOS Keychain service name")
		keychainAcct = flag.String("keychain-account", firstNonEmpty(os.Getenv("EO_UPDATE_PRIVATE_KEY_KEYCHAIN_ACCOUNT"), os.Getenv("USER")), "macOS Keychain account name")
	)
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "--file is required")
		os.Exit(1)
	}
	key, err := resolvePrivateKey(*privateKey, *keychainSvc, *keychainAcct)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

func resolvePrivateKey(flagValue, keychainService, keychainAccount string) (string, error) {
	if key := firstNonEmpty(flagValue, os.Getenv("EO_UPDATE_PRIVATE_KEY")); key != "" {
		return key, nil
	}
	if runtime.GOOS != "darwin" {
		return "", nil
	}
	if strings.TrimSpace(keychainService) == "" || strings.TrimSpace(keychainAccount) == "" {
		return "", nil
	}
	cmd := exec.Command("security", "find-generic-password", "-a", keychainAccount, "-s", keychainService, "-w")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("private key not provided and macOS Keychain lookup failed for service %q", keychainService)
	}
	return strings.TrimSpace(string(output)), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
