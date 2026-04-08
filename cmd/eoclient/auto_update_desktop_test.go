//go:build !js

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestShouldApplyBinaryUpdate(t *testing.T) {
	tests := []struct {
		current string
		target  string
		want    bool
	}{
		{current: "v0.1.0", target: "v0.1.1", want: true},
		{current: "v0.2.0", target: "v0.1.9", want: false},
		{current: "v1.0.0", target: "v1.0.0", want: false},
		{current: "dev", target: "v1.0.0", want: false},
	}
	for _, tc := range tests {
		if got := shouldApplyBinaryUpdate(tc.current, tc.target); got != tc.want {
			t.Fatalf("shouldApplyBinaryUpdate(%q, %q) = %v, want %v", tc.current, tc.target, got, tc.want)
		}
	}
}

func TestIsSemverVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{version: "v0.1.0", want: true},
		{version: "1.2.3", want: true},
		{version: "v1.2", want: true},
		{version: "dev", want: false},
		{version: "abc1234", want: false},
		{version: "release-test", want: false},
	}
	for _, tc := range tests {
		if got := isSemverVersion(tc.version); got != tc.want {
			t.Fatalf("isSemverVersion(%q) = %v, want %v", tc.version, got, tc.want)
		}
	}
}

func TestExtractArchiveBinaryTarGz(t *testing.T) {
	var payload bytes.Buffer
	gzipWriter := gzip.NewWriter(&payload)
	tarWriter := tar.NewWriter(gzipWriter)
	data := []byte("binary")
	header := &tar.Header{Name: "geoclient", Mode: 0o755, Size: int64(len(data))}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	if _, err := tarWriter.Write(data); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path, err := extractArchiveBinary(payload.Bytes(), false)
	if err != nil {
		t.Fatalf("extractArchiveBinary() error = %v", err)
	}
	if string(path) != "binary" {
		t.Fatalf("extractArchiveBinary() = %q, want binary", string(path))
	}
}

func TestExtractArchiveBinaryZip(t *testing.T) {
	var payload bytes.Buffer
	zipWriter := zip.NewWriter(&payload)
	writer, err := zipWriter.Create("geoclient.exe")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := writer.Write([]byte("binary")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path, err := extractArchiveBinary(payload.Bytes(), true)
	if err != nil {
		t.Fatalf("extractArchiveBinary() error = %v", err)
	}
	if string(path) != "binary" {
		t.Fatalf("extractArchiveBinary() = %q, want binary", string(path))
	}
}
