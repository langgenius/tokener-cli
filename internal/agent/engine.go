package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/langgenius/tokener-cli/internal/atomicfile"
	"github.com/langgenius/tokener-cli/internal/rxsnapshot"
)

type embeddedEngine struct {
	data        []byte
	digest      string
	version     string
	revision    string
	targetOS    string
	targetArch  string
	metadataErr error
	cacheRoot   func() (string, error)
	lookupEnv   func(string) (string, bool)
}

func newEmbeddedEngine() embeddedEngine {
	version, revision, digest, metadataErr := loadEmbeddedRXMetadata(embeddedRXOS, embeddedRXArch)
	return embeddedEngine{
		data:        embeddedRX,
		digest:      digest,
		version:     version,
		revision:    revision,
		targetOS:    embeddedRXOS,
		targetArch:  embeddedRXArch,
		metadataErr: metadataErr,
		cacheRoot:   cacheDirectory,
		lookupEnv:   os.LookupEnv,
	}
}

func (engine embeddedEngine) Resolve(ctx context.Context, harness string) (string, error) {
	if override, configured := engine.lookupEnv("TOKENER_RX"); configured {
		if override == "" {
			return "", errors.New("TOKENER_RX is empty")
		}
		if !filepath.IsAbs(override) {
			return "", errors.New("TOKENER_RX must be an absolute path")
		}
		if err := validateOverride(ctx, override, harness); err != nil {
			return "", err
		}
		return override, nil
	}
	if len(engine.data) == 0 {
		return "", fmt.Errorf("embedded rx is unavailable for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if engine.metadataErr != nil {
		return "", fmt.Errorf("load embedded rx metadata: %w", engine.metadataErr)
	}
	if engine.targetOS != runtime.GOOS || engine.targetArch != runtime.GOARCH {
		return "", fmt.Errorf("embedded rx targets %s/%s, not %s/%s", engine.targetOS, engine.targetArch, runtime.GOOS, runtime.GOARCH)
	}
	digest := rxsnapshot.Digest(engine.data)
	if digest != engine.digest {
		return "", fmt.Errorf(
			"embedded rx %s from %s failed SHA-256 verification",
			engine.version,
			engine.revision,
		)
	}
	root, err := engine.cacheRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "engines", digest)
	name := "rx"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if engineFileMatches(path, digest) {
		return path, nil
	}
	if err := atomicfile.PrivateDir(dir); err != nil {
		return "", fmt.Errorf("create rx engine cache: %w", err)
	}
	if err := atomicfile.Write(path, engine.data, 0o700); err != nil {
		var renameError *os.LinkError
		if !errors.As(err, &renameError) || !engineFileMatches(path, digest) {
			return "", fmt.Errorf("install rx engine: %w", err)
		}
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", fmt.Errorf("make rx engine executable: %w", err)
	}
	if !engineFileMatches(path, digest) {
		return "", errors.New("released rx engine failed SHA-256 verification")
	}
	return path, nil
}

func agentStateDir() (string, error) {
	root, err := cacheDirectory()
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "agent")
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve Tokener agent state directory: %w", err)
	}
	return absolute, nil
}

func cacheDirectory() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve Tokener cache directory: %w", err)
	}
	return filepath.Join(root, "tokener"), nil
}

func engineFileMatches(path, digest string) bool {
	data, err := os.ReadFile(path)
	return err == nil && rxsnapshot.Digest(data) == digest
}
