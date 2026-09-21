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

	if err := binding.Save(hostname, bindingDocument{Key: "first-key", KeyID: "key-1"}); err != nil {
		t.Fatal(err)
	}
	document, exists, err := binding.Load(hostname)
	if err != nil || !exists || document != (bindingDocument{Key: "first-key", KeyID: "key-1"}) {
		t.Fatalf("load first key = %#v/%t/%v", document, exists, err)
	}
	if err := binding.Save(hostname, bindingDocument{Key: "second-key", KeyID: "key-2"}); err != nil {
		t.Fatal(err)
	}
	document, exists, err = binding.Load(hostname)
	if err != nil || !exists || document != (bindingDocument{Key: "second-key", KeyID: "key-2"}) {
		t.Fatalf("load second key = %#v/%t/%v", document, exists, err)
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
	if err := binding.Save("console.tokener.dev", bindingDocument{Key: "prod-key"}); err != nil {
		t.Fatal(err)
	}
	if err := binding.Save("http://localhost:3000", bindingDocument{Key: "local-key"}); err != nil {
		t.Fatal(err)
	}
	document, exists, err := binding.Load("console.tokener.dev")
	if err != nil || !exists || document.Key != "prod-key" {
		t.Fatalf("prod key = %#v/%t/%v", document, exists, err)
	}
	document, exists, err = binding.Load("http://localhost:3000")
	if err != nil || !exists || document.Key != "local-key" {
		t.Fatalf("local key = %#v/%t/%v", document, exists, err)
	}

	legacy := filepath.Join(dir, "agent-key.json")
	if err := os.WriteFile(legacy, []byte(`{"key":"legacy-key"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	isolated := fileBinding{}
	if err := os.RemoveAll(filepath.Join(dir, "agent-keys")); err != nil {
		t.Fatal(err)
	}
	document, exists, err = isolated.Load(defaultManagementHostname)
	if err != nil || !exists || document != (bindingDocument{Key: "legacy-key"}) {
		t.Fatalf("legacy fallback = %#v/%t/%v", document, exists, err)
	}
	document, exists, err = isolated.Load("console-staging.tokener.dev")
	if err != nil || exists || document.Key != "" {
		t.Fatalf("staging should not use legacy = %#v/%t/%v", document, exists, err)
	}
}

func TestFileBindingRejectsEmptyAndMalformedDocuments(t *testing.T) {
	bindAgentTestManifest(t)
	binding := fileBinding{}
	if err := binding.Save(defaultManagementHostname, bindingDocument{}); err == nil {
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
