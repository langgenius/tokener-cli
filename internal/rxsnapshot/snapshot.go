package rxsnapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const Schema = 1

type Target struct {
	Key    string
	Path   string
	GOOS   string
	GOARCH string
}

type Source struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
	Revision   string `json:"revision"`
	Version    string `json:"version"`
}

type Build struct {
	RustToolchain string `json:"rust_toolchain"`
	Provenance    string `json:"provenance"`
}

type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Snapshot struct {
	Schema    int                 `json:"schema"`
	Source    Source              `json:"source"`
	Build     Build               `json:"build"`
	Artifacts map[string]Artifact `json:"artifacts"`
}

var targets = []Target{
	{Key: "darwin/amd64", Path: "internal/agent/assets/rx-darwin-amd64", GOOS: "darwin", GOARCH: "amd64"},
	{Key: "darwin/arm64", Path: "internal/agent/assets/rx", GOOS: "darwin", GOARCH: "arm64"},
	{Key: "linux/amd64", Path: "internal/agent/assets/rx-linux-amd64", GOOS: "linux", GOARCH: "amd64"},
	{Key: "windows/amd64", Path: "internal/agent/assets/rx-windows-amd64.exe", GOOS: "windows", GOARCH: "amd64"},
}

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func Targets() []Target {
	result := make([]Target, len(targets))
	copy(result, targets)
	return result
}

func Parse(contents []byte) (Snapshot, error) {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var snapshot Snapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("parse rx snapshot: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Snapshot{}, errors.New("parse rx snapshot: trailing JSON value")
	}
	if err := snapshot.Validate(); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func Read(path string) (Snapshot, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read rx snapshot %s: %w", path, err)
	}
	return Parse(contents)
}

func (snapshot Snapshot) Validate() error {
	if snapshot.Schema != Schema {
		return fmt.Errorf("rx snapshot schema = %d, expected %d", snapshot.Schema, Schema)
	}
	if strings.TrimSpace(snapshot.Source.Repository) == "" {
		return errors.New("rx snapshot repository is empty")
	}
	if strings.TrimSpace(snapshot.Source.Ref) == "" {
		return errors.New("rx snapshot ref is empty")
	}
	if !revisionPattern.MatchString(snapshot.Source.Revision) {
		return errors.New("rx snapshot revision must be a full lowercase commit SHA")
	}
	if strings.TrimSpace(snapshot.Source.Version) == "" {
		return errors.New("rx snapshot version is empty")
	}
	if strings.TrimSpace(snapshot.Build.RustToolchain) == "" {
		return errors.New("rx snapshot Rust toolchain is empty")
	}
	if !strings.HasPrefix(snapshot.Build.Provenance, "https://github.com/") {
		return errors.New("rx snapshot provenance must be a GitHub URL")
	}
	if len(snapshot.Artifacts) != len(targets) {
		return fmt.Errorf("rx snapshot artifact count = %d, expected %d", len(snapshot.Artifacts), len(targets))
	}
	for _, target := range targets {
		artifact, ok := snapshot.Artifacts[target.Key]
		if !ok {
			return fmt.Errorf("rx snapshot is missing %s", target.Key)
		}
		if artifact.Path != target.Path {
			return fmt.Errorf("rx snapshot path for %s = %q, expected %q", target.Key, artifact.Path, target.Path)
		}
		if !digestPattern.MatchString(artifact.SHA256) {
			return fmt.Errorf("rx snapshot SHA-256 for %s is invalid", target.Key)
		}
	}
	return nil
}

func (snapshot Snapshot) Artifact(goos, goarch string) (Artifact, bool) {
	artifact, ok := snapshot.Artifacts[goos+"/"+goarch]
	return artifact, ok
}

func (snapshot Snapshot) VerifyFiles(root string) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	for _, target := range targets {
		artifact := snapshot.Artifacts[target.Key]
		path := filepath.Join(root, filepath.FromSlash(artifact.Path))
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("inspect rx artifact %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("rx artifact %s is not a regular file", path)
		}
		digest, err := fileSHA256(path)
		if err != nil {
			return err
		}
		if digest != artifact.SHA256 {
			return fmt.Errorf("rx artifact %s SHA-256 = %s, expected %s", target.Key, digest, artifact.SHA256)
		}
	}
	return nil
}

func New(root string, source Source, build Build) (Snapshot, error) {
	snapshot := Snapshot{
		Schema:    Schema,
		Source:    source,
		Build:     build,
		Artifacts: make(map[string]Artifact, len(targets)),
	}
	for _, target := range targets {
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		digest, err := fileSHA256(path)
		if err != nil {
			return Snapshot{}, err
		}
		snapshot.Artifacts[target.Key] = Artifact{Path: target.Path, SHA256: digest}
	}
	if err := snapshot.Validate(); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func Write(path string, snapshot Snapshot) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rx snapshot: %w", err)
	}
	contents = append(contents, '\n')
	dir := filepath.Dir(path)
	temporary, err := os.CreateTemp(dir, ".rx-lock-*.tmp")
	if err != nil {
		return fmt.Errorf("create rx snapshot: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("set rx snapshot mode: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write rx snapshot: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync rx snapshot: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close rx snapshot: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace rx snapshot: %w", err)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read rx artifact %s: %w", path, err)
	}
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:]), nil
}
