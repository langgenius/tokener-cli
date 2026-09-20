package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/langgenius/tokener-cli/internal/rxsnapshot"
)

func tarGz(t *testing.T, members map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	compressor := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(compressor)
	for name, contents := range members {
		header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(contents)), Typeflag: tar.TypeReg}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressor.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func zipped(t *testing.T, members map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, contents := range members {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestExtractSelectsRXFromReleaseArchives(t *testing.T) {
	archive := tarGz(t, map[string]string{"recall": "recall-binary", "rx": "rx-binary"})
	contents, err := extract("recall-macos-aarch64.tar.gz", "rx", archive)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "rx-binary" {
		t.Fatalf("extracted %q", contents)
	}

	windows := zipped(t, map[string]string{"recall.exe": "recall-binary", "rx.exe": "rx-binary"})
	contents, err = extract("recall-windows-x86_64.zip", "rx.exe", windows)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "rx-binary" {
		t.Fatalf("extracted %q", contents)
	}
}

func TestExtractRejectsArchiveWithoutRX(t *testing.T) {
	if _, err := extract("recall-linux-x86_64.tar.gz", "rx", tarGz(t, map[string]string{"recall": "recall-binary"})); err == nil {
		t.Fatal("archive without rx was accepted")
	}
	if _, err := extract("recall-windows-x86_64.zip", "rx.exe", zipped(t, map[string]string{"recall.exe": "recall-binary"})); err == nil {
		t.Fatal("archive without rx.exe was accepted")
	}
}

func TestParseWorkspaceVersionIgnoresPackageVersions(t *testing.T) {
	manifest := []byte(`[package]
name = "recall"
version.workspace = true

[workspace]
members = [".", "crates/rx"]

[workspace.package]
version = "0.6.1"

[dependencies]
clap = { version = "4" }
`)
	version, ok := parseWorkspaceVersion(manifest)
	if !ok || version != "0.6.1" {
		t.Fatalf("version = %q, ok = %v", version, ok)
	}
	if _, ok := parseWorkspaceVersion([]byte("[package]\nversion = \"9.9.9\"\n")); ok {
		t.Fatal("a manifest without a workspace package version was accepted")
	}
}

func TestReleaseArtifactsCoverEverySnapshotTarget(t *testing.T) {
	seen := make(map[string]bool, len(releaseArtifacts))
	for _, artifact := range releaseArtifacts {
		if _, ok := rxsnapshot.Path(artifact.key); !ok {
			t.Fatalf("release artifact %s has no snapshot target", artifact.asset)
		}
		if seen[artifact.key] {
			t.Fatalf("release artifact %s duplicates %s", artifact.asset, artifact.key)
		}
		seen[artifact.key] = true
	}
	for _, key := range []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "windows/amd64"} {
		if !seen[key] {
			t.Fatalf("no release artifact provides %s", key)
		}
	}
}
