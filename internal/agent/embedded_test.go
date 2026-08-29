package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestEmbeddedRXMatchesSnapshotAndHostedProtocol(t *testing.T) {
	engine := newEmbeddedEngine()
	if len(engine.data) == 0 {
		t.Skip("embedded rx is unavailable on this platform")
	}
	if engine.metadataErr != nil {
		t.Fatal(engine.metadataErr)
	}
	if digest := sha256Hex(engine.data); digest != engine.digest {
		t.Fatalf("embedded rx SHA-256 = %s", digest)
	}
	if len(engine.revision) != 40 {
		t.Fatalf("embedded rx revision = %q", engine.revision)
	}
	name := "rx"
	if engine.targetOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, engine.data, 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(path, "host").Output()
	if err != nil {
		t.Fatal(err)
	}
	var response capabilities
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatal(err)
	}
	if response.Protocol.Major != 1 || response.Protocol.Minor != 0 {
		t.Fatalf("protocol = %d.%d", response.Protocol.Major, response.Protocol.Minor)
	}
	if response.Version != engine.version {
		t.Fatalf("version = %q", response.Version)
	}
	if !slices.Equal(response.Harnesses, harnesses) {
		t.Fatalf("harnesses = %v", response.Harnesses)
	}
}
