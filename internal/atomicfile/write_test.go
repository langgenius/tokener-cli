package atomicfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteReplacesAndCleansUpOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	for _, body := range []string{"first", "second"} {
		if err := Write(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		contents, err := os.ReadFile(path)
		if err != nil || string(contents) != body {
			t.Fatalf("contents = %q, error = %v", contents, err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if err := Write(dir, []byte("cannot replace directory"), 0o600); err == nil {
		t.Fatal("replaced directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "key" {
		t.Fatalf("entries = %v, error = %v", entries, err)
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(dir), "."+filepath.Base(dir)+"-*.tmp"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary files = %v, error = %v", leftovers, err)
	}
}
