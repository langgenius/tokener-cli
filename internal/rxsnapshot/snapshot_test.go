package rxsnapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotDetectsArtifactDrift(t *testing.T) {
	root := t.TempDir()
	for _, target := range Targets() {
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(target.Key), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := New(
		root,
		Source{
			Repository: "samzong/Recall",
			Ref:        "main",
			Revision:   "a45c10e18d4fdf1d15e4b1b3fb11365480750618",
			Version:    "0.5.7",
		},
		Build{
			RustToolchain: "1.96.0",
			Provenance:    "https://github.com/langgenius/tokener-cli/actions/runs/1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.VerifyFiles(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(Targets()[0].Path))
	if err := os.WriteFile(path, []byte("changed"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.VerifyFiles(root); err == nil {
		t.Fatal("artifact drift was accepted")
	}
}
