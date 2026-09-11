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
	binding := newFileBinding()
	hostname := "console-staging.tokener.dev"

	if err := binding.Save(hostname, "first-key"); err != nil {
		t.Fatal(err)
	}
	key, exists, err := binding.Load(hostname)
	if err != nil || !exists || key != "first-key" {
		t.Fatalf("load first key = %q/%t/%v", key, exists, err)
	}
	if err := binding.Save(hostname, "second-key"); err != nil {
		t.Fatal(err)
	}
	key, exists, err = binding.Load(hostname)
	if err != nil || !exists || key != "second-key" {
		t.Fatalf("load second key = %q/%t/%v", key, exists, err)
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
	binding := newFileBinding()
	if err := binding.Save("console.tokener.dev", "prod-key"); err != nil {
		t.Fatal(err)
	}
	if err := binding.Save("http://localhost:3000", "local-key"); err != nil {
		t.Fatal(err)
	}
	key, exists, err := binding.Load("console.tokener.dev")
	if err != nil || !exists || key != "prod-key" {
		t.Fatalf("prod key = %q/%t/%v", key, exists, err)
	}
	key, exists, err = binding.Load("http://localhost:3000")
	if err != nil || !exists || key != "local-key" {
		t.Fatalf("local key = %q/%t/%v", key, exists, err)
	}

	legacy := filepath.Join(dir, "agent-key.json")
	if err := os.WriteFile(legacy, []byte(`{"key":"legacy-key"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	isolated := newFileBinding()
	if err := os.RemoveAll(filepath.Join(dir, "agent-keys")); err != nil {
		t.Fatal(err)
	}
	key, exists, err = isolated.Load(defaultManagementHostname)
	if err != nil || !exists || key != "legacy-key" {
		t.Fatalf("legacy fallback = %q/%t/%v", key, exists, err)
	}
	key, exists, err = isolated.Load("console-staging.tokener.dev")
	if err != nil || exists || key != "" {
		t.Fatalf("staging should not use legacy = %q/%t/%v", key, exists, err)
	}
}

func TestFileBindingRejectsEmptyAndMalformedDocuments(t *testing.T) {
	bindAgentTestManifest(t)
	binding := newFileBinding()
	if err := binding.Save(defaultManagementHostname, ""); err == nil {
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
