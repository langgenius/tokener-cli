package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/langgenius/tokener-cli/internal/atomicfile"
	"github.com/langgenius/tokener-cli/internal/rxsnapshot"
)

type releaseArtifact struct {
	asset  string
	member string
	key    string
}

var releaseArtifacts = []releaseArtifact{
	{asset: "recall-macos-x86_64.tar.gz", member: "rx", key: "darwin/amd64"},
	{asset: "recall-macos-aarch64.tar.gz", member: "rx", key: "darwin/arm64"},
	{asset: "recall-linux-x86_64.tar.gz", member: "rx", key: "linux/amd64"},
	{asset: "recall-windows-x86_64.zip", member: "rx.exe", key: "windows/amd64"},
}

var client = &http.Client{Timeout: 5 * time.Minute}

func pull(root, manifest, repository, tag, toolchain string) error {
	revision, err := resolveRevision(repository, tag)
	if err != nil {
		return err
	}
	version, err := resolveVersion(repository, revision)
	if err != nil {
		return err
	}
	for _, artifact := range releaseArtifacts {
		target, ok := rxsnapshot.Path(artifact.key)
		if !ok {
			return fmt.Errorf("rx snapshot has no target for %s", artifact.key)
		}
		archive, err := fetch(fmt.Sprintf(
			"https://github.com/%s/releases/download/%s/%s",
			repository,
			url.PathEscape(tag),
			url.PathEscape(artifact.asset),
		))
		if err != nil {
			return err
		}
		binary, err := extract(artifact.asset, artifact.member, archive)
		if err != nil {
			return err
		}
		destination := filepath.Join(root, filepath.FromSlash(target))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("create rx asset directory: %w", err)
		}
		if err := atomicfile.Write(destination, binary, 0o755); err != nil {
			return fmt.Errorf("install %s: %w", target, err)
		}
	}
	snapshot, err := rxsnapshot.New(
		root,
		rxsnapshot.Source{
			Repository: repository,
			Ref:        tag,
			Revision:   revision,
			Version:    version,
		},
		rxsnapshot.Build{
			RustToolchain: toolchain,
			Provenance:    fmt.Sprintf("https://github.com/%s/releases/tag/%s", repository, tag),
		},
	)
	if err != nil {
		return err
	}
	if err := rxsnapshot.Write(manifest, snapshot); err != nil {
		return err
	}
	if err := snapshot.VerifyFiles(root); err != nil {
		return err
	}
	fmt.Printf("rx snapshot %s@%s pulled from %s\n", repository, revision, tag)
	return nil
}

func resolveRevision(repository, tag string) (string, error) {
	object, err := gitObject(fmt.Sprintf(
		"https://api.github.com/repos/%s/git/ref/tags/%s",
		repository,
		url.PathEscape(tag),
	))
	if err != nil {
		return "", err
	}
	if object.Type == "tag" {
		object, err = gitObject(fmt.Sprintf("https://api.github.com/repos/%s/git/tags/%s", repository, object.SHA))
		if err != nil {
			return "", err
		}
	}
	if object.Type != "commit" {
		return "", fmt.Errorf("tag %s resolves to a %s, not a commit", tag, object.Type)
	}
	return object.SHA, nil
}

var workspaceVersionPattern = regexp.MustCompile(`(?ms)^\[workspace\.package\][^\[]*?^version\s*=\s*"([^"]+)"`)

func resolveVersion(repository, revision string) (string, error) {
	manifest, err := fetch(fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/Cargo.toml", repository, revision))
	if err != nil {
		return "", err
	}
	version, ok := parseWorkspaceVersion(manifest)
	if !ok {
		return "", fmt.Errorf("%s@%s declares no workspace package version", repository, revision)
	}
	return version, nil
}

func parseWorkspaceVersion(manifest []byte) (string, bool) {
	match := workspaceVersionPattern.FindSubmatch(manifest)
	if match == nil {
		return "", false
	}
	return string(match[1]), true
}

type gitReference struct {
	SHA  string `json:"sha"`
	Type string `json:"type"`
}

func gitObject(endpoint string) (gitReference, error) {
	body, err := fetch(endpoint)
	if err != nil {
		return gitReference{}, err
	}
	var payload struct {
		Object gitReference `json:"object"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return gitReference{}, fmt.Errorf("decode %s: %w", endpoint, err)
	}
	if payload.Object.SHA == "" {
		return gitReference{}, fmt.Errorf("%s returned no object", endpoint)
	}
	return payload.Object, nil
}

func fetch(endpoint string) ([]byte, error) {
	response, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", endpoint, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %s", endpoint, response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", endpoint, err)
	}
	return body, nil
}

func extract(asset, member string, archive []byte) ([]byte, error) {
	if strings.HasSuffix(asset, ".zip") {
		return extractZip(asset, member, archive)
	}
	return extractTar(asset, member, archive)
}

func extractTar(asset, member string, archive []byte) ([]byte, error) {
	decompressed, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", asset, err)
	}
	defer decompressed.Close()
	reader := tar.NewReader(decompressed)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", asset, err)
		}
		if header.Typeflag != tar.TypeReg || path.Base(header.Name) != member {
			continue
		}
		contents, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read %s from %s: %w", member, asset, err)
		}
		return contents, nil
	}
	return nil, fmt.Errorf("%s contains no %s", asset, member)
}

func extractZip(asset, member string, archive []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", asset, err)
	}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || path.Base(file.Name) != member {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("read %s from %s: %w", member, asset, err)
		}
		defer entry.Close()
		contents, err := io.ReadAll(entry)
		if err != nil {
			return nil, fmt.Errorf("read %s from %s: %w", member, asset, err)
		}
		return contents, nil
	}
	return nil, fmt.Errorf("%s contains no %s", asset, member)
}
