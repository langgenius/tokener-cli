package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileBindingSavesLoadsAndReplacesAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "agent-key.json")
	binding := fileBinding{path: func() (string, error) { return path, nil }}

	if err := binding.Save("first-key"); err != nil {
		t.Fatal(err)
	}
	key, exists, err := binding.Load()
	if err != nil || !exists || key != "first-key" {
		t.Fatalf("load first key = %q/%t/%v", key, exists, err)
	}
	if err := binding.Save("second-key"); err != nil {
		t.Fatal(err)
	}
	key, exists, err = binding.Load()
	if err != nil || !exists || key != "second-key" {
		t.Fatalf("load second key = %q/%t/%v", key, exists, err)
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

func TestFileBindingRejectsEmptyAndMalformedDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-key.json")
	binding := fileBinding{path: func() (string, error) { return path, nil }}
	if err := binding.Save(""); err == nil {
		t.Fatal("empty key was accepted")
	}
	if err := os.WriteFile(path, []byte(`{"key":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := binding.Load(); err == nil {
		t.Fatal("empty binding was accepted")
	}
	if err := os.WriteFile(path, []byte(`{"key":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := binding.Load(); err == nil {
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
