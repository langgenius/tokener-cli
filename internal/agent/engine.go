package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const requestEnvironment = "RX_HOST_REQUEST"

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

type hostRequest struct {
	Harness          string         `json:"harness,omitempty"`
	Gateway          gatewayProfile `json:"gateway"`
	StateDir         string         `json:"state_dir"`
	PermissionPolicy string         `json:"permission_policy"`
	InstallPolicy    string         `json:"install_policy"`
}

type gatewayProfile struct {
	ProviderID    string `json:"provider_id"`
	Name          string `json:"name"`
	Endpoint      string `json:"endpoint"`
	CredentialEnv string `json:"credential_env"`
}

type capabilities struct {
	Protocol struct {
		Major int `json:"major"`
		Minor int `json:"minor"`
	} `json:"protocol"`
	Version   string   `json:"version"`
	Harnesses []string `json:"harnesses"`
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
	digest := sha256Hex(engine.data)
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create rx engine cache: %w", err)
	}
	if err := restrictDirectory(dir); err != nil {
		return "", fmt.Errorf("restrict rx engine cache: %w", err)
	}
	temporary, err := os.CreateTemp(dir, ".rx-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary rx engine: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o700); err != nil {
		temporary.Close()
		return "", fmt.Errorf("make temporary rx engine executable: %w", err)
	}
	if _, err := temporary.Write(engine.data); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write temporary rx engine: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", fmt.Errorf("sync temporary rx engine: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close temporary rx engine: %w", err)
	}
	if err := replaceFile(temporaryPath, path); err != nil && !engineFileMatches(path, digest) {
		return "", fmt.Errorf("install rx engine: %w", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", fmt.Errorf("make rx engine executable: %w", err)
	}
	if !engineFileMatches(path, digest) {
		return "", errors.New("released rx engine failed SHA-256 verification")
	}
	return path, nil
}

func validateOverride(ctx context.Context, path, harness string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect TOKENER_RX: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("TOKENER_RX must point to a regular file")
	}
	command := exec.CommandContext(ctx, path, "host")
	command.Env = environmentWithout(os.Environ(), requestEnvironment)
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("TOKENER_RX host handshake failed: %w", err)
	}
	var response capabilities
	if err := json.Unmarshal(output, &response); err != nil {
		return fmt.Errorf("TOKENER_RX host handshake returned invalid JSON: %w", err)
	}
	if response.Protocol.Major != 1 || response.Protocol.Minor < 0 {
		return fmt.Errorf("TOKENER_RX host protocol %d.%d is incompatible with 1.0", response.Protocol.Major, response.Protocol.Minor)
	}
	required := harnesses
	if harness != "" {
		required = []string{harness}
	}
	for _, name := range required {
		if !slices.Contains(response.Harnesses, name) {
			return fmt.Errorf("TOKENER_RX host does not support %s", name)
		}
	}
	return nil
}

func launchEngine(path string, request hostRequest, nativeArgs []string, key string) error {
	args, environment, err := launchSpec(request, nativeArgs, key, os.Environ())
	if err != nil {
		return err
	}
	return execProcess(path, args, environment)
}

func launchSpec(request hostRequest, nativeArgs []string, key string, baseEnvironment []string) ([]string, []string, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, nil, fmt.Errorf("encode rx host request: %w", err)
	}
	args := []string{"host", "--"}
	args = append(args, nativeArgs...)
	environment := environmentWithout(baseEnvironment, requestEnvironment, credentialEnv)
	environment = append(environment, requestEnvironment+"="+string(payload), credentialEnv+"="+key)
	return args, environment, nil
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
	return err == nil && sha256Hex(data) == digest
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func environmentWithout(environment []string, names ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		name := strings.SplitN(entry, "=", 2)[0]
		if !slices.Contains(names, name) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
