//go:build darwin && arm64

package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestEmbeddedRXMatchesPinnedHashAndHostedProtocol(t *testing.T) {
	if digest := sha256Hex(embeddedRX); digest != embeddedRXSHA256 {
		t.Fatalf("embedded rx SHA-256 = %s", digest)
	}
	path := filepath.Join(t.TempDir(), "rx")
	if err := os.WriteFile(path, embeddedRX, 0o700); err != nil {
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
	if response.Version != embeddedRXVersion {
		t.Fatalf("version = %q", response.Version)
	}
	if !slices.Equal(response.Harnesses, harnesses) {
		t.Fatalf("harnesses = %v", response.Harnesses)
	}
}
