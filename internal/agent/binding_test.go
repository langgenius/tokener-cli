package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lathe-cli/lathe/pkg/config"
)

func bindAgentTestManifest(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("TOKENER_CONFIG_DIR", root)
	config.Bind(&config.Manifest{CLI: config.CLIInfo{
		Name:         "tokener",
		ConfigDir:    "tokener",
		ConfigDirEnv: "TOKENER_CONFIG_DIR",
		HostEnv:      "TOKENER_HOST",
	}})
	return filepath.Join(root, "tokener")
}

func TestFileBindingSavesLoadsAndReplacesAtomically(t *testing.T) {
	bindAgentTestManifest(t)
	binding := fileBinding{}
	hostname := "console-staging.tokener.dev"

	first := agentBinding{Key: "first-key", KeyID: "key-1", Name: "Tokener Agent CLI · box-abc123", MachineID: "abc123"}
	if err := binding.Save(hostname, first); err != nil {
		t.Fatal(err)
	}
	stored, exists, err := binding.Load(hostname)
	if err != nil || !exists || stored != first {
		t.Fatalf("load first key = %#v/%t/%v", stored, exists, err)
	}
	second := agentBinding{Key: "second-key", KeyID: "key-2", Name: "Tokener Agent CLI · box-abc123", MachineID: "abc123"}
	if err := binding.Save(hostname, second); err != nil {
		t.Fatal(err)
	}
	stored, exists, err = binding.Load(hostname)
	if err != nil || !exists || stored != second {
		t.Fatalf("load second key = %#v/%t/%v", stored, exists, err)
	}
	path, err := bindingPathFor(hostname)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		mode := fileMode(t, path)
		if mode != 0o600 {
			t.Fatalf("binding mode = %o", mode)
		}
		if directoryMode := fileMode(t, filepath.Dir(path)); directoryMode != 0o700 {
			t.Fatalf("binding directory mode = %o", directoryMode)
		}
	}
}

func TestFileBindingIsolatesHostsAndFallsBackToLegacyDefault(t *testing.T) {
	dir := bindAgentTestManifest(t)
	binding := fileBinding{}
	if err := binding.Save("console.tokener.dev", agentBinding{Key: "prod-key"}); err != nil {
		t.Fatal(err)
	}
	if err := binding.Save("http://localhost:3000", agentBinding{Key: "local-key"}); err != nil {
		t.Fatal(err)
	}
	stored, exists, err := binding.Load("console.tokener.dev")
	if err != nil || !exists || stored.Key != "prod-key" {
		t.Fatalf("prod key = %#v/%t/%v", stored, exists, err)
	}
	stored, exists, err = binding.Load("http://localhost:3000")
	if err != nil || !exists || stored.Key != "local-key" {
		t.Fatalf("local key = %#v/%t/%v", stored, exists, err)
	}

	legacy := filepath.Join(dir, "agent-key.json")
	if err := os.WriteFile(legacy, []byte(`{"key":"legacy-key"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	isolated := fileBinding{}
	if err := os.RemoveAll(filepath.Join(dir, "agent-keys")); err != nil {
		t.Fatal(err)
	}
	stored, exists, err = isolated.Load(defaultManagementHostname)
	if err != nil || !exists || stored.Key != "legacy-key" {
		t.Fatalf("legacy fallback = %#v/%t/%v", stored, exists, err)
	}
	// Bindings predating per-machine names carry no id, name or machine id.
	if stored.KeyID != "" || stored.Name != "" || stored.MachineID != "" {
		t.Fatalf("legacy binding gained fields = %#v", stored)
	}
	stored, exists, err = isolated.Load("console-staging.tokener.dev")
	if err != nil || exists || stored.Key != "" {
		t.Fatalf("staging should not use legacy = %#v/%t/%v", stored, exists, err)
	}
}

func TestFileBindingRejectsEmptyAndMalformedDocuments(t *testing.T) {
	bindAgentTestManifest(t)
	binding := fileBinding{}
	if err := binding.Save(defaultManagementHostname, agentBinding{}); err == nil {
		t.Fatal("empty key was accepted")
	}
	path, err := bindingPathFor(defaultManagementHostname)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"key":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := binding.Load(defaultManagementHostname); err == nil {
		t.Fatal("empty binding was accepted")
	}
	if err := os.WriteFile(path, []byte(`{"key":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := binding.Load(defaultManagementHostname); err == nil {
		t.Fatal("malformed binding was accepted")
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
