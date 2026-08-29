package agent

import (
	_ "embed"
	"fmt"

	"github.com/langgenius/tokener-cli/internal/rxsnapshot"
)

//go:embed rx.lock.json
var embeddedRXLock []byte

func loadEmbeddedRXMetadata(goos, goarch string) (string, string, string, error) {
	snapshot, err := rxsnapshot.Parse(embeddedRXLock)
	if err != nil {
		return "", "", "", err
	}
	artifact, ok := snapshot.Artifact(goos, goarch)
	if !ok {
		return "", "", "", fmt.Errorf("rx snapshot has no artifact for %s/%s", goos, goarch)
	}
	return snapshot.Source.Version, snapshot.Source.Revision, artifact.SHA256, nil
}
