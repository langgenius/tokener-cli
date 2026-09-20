package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/langgenius/tokener-cli/internal/atomicfile"
	"github.com/lathe-cli/lathe/pkg/config"
)

type fileBinding struct{}

// agentBinding is the locally stored description of this machine's agent key.
// KeyID, Name and MachineID are absent in bindings written before per-machine
// key names existed; callers must tolerate empty values.
type agentBinding struct {
	Key       string `json:"key"`
	KeyID     string `json:"keyId,omitempty"`
	Name      string `json:"name,omitempty"`
	MachineID string `json:"machineId,omitempty"`
}

func (binding fileBinding) Load(hostname string) (agentBinding, bool, error) {
	path, err := bindingPathFor(hostname)
	if err != nil {
		return agentBinding{}, false, err
	}
	stored, exists, err := loadBindingFile(path)
	if err != nil || exists {
		return stored, exists, err
	}
	if config.NormalizeHostname(hostname) != defaultManagementHostname {
		return agentBinding{}, false, nil
	}
	return loadBindingFile(filepath.Join(filepath.Dir(filepath.Dir(path)), "agent-key.json"))
}

func (binding fileBinding) Save(hostname string, stored agentBinding) error {
	if strings.TrimSpace(stored.Key) == "" {
		return errors.New("agent key is empty")
	}
	path, err := bindingPathFor(hostname)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := atomicfile.PrivateDir(dir); err != nil {
		return fmt.Errorf("create agent config directory: %w", err)
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("encode agent key binding: %w", err)
	}
	if err := atomicfile.Write(path, data, 0o600); err != nil {
		return fmt.Errorf("write agent key binding: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict agent key binding: %w", err)
	}
	return nil
}

func loadBindingFile(path string) (agentBinding, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return agentBinding{}, false, nil
	}
	if err != nil {
		return agentBinding{}, false, fmt.Errorf("read agent key binding: %w", err)
	}
	var document agentBinding
	if err := json.Unmarshal(data, &document); err != nil {
		return agentBinding{}, false, fmt.Errorf("parse agent key binding: %w", err)
	}
	if strings.TrimSpace(document.Key) == "" {
		return agentBinding{}, false, errors.New("agent key binding is empty")
	}
	return document, true, nil
}

func bindingPathFor(hostname string) (string, error) {
	dir, err := configDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "agent-keys", bindingFileName(hostname)), nil
}

func bindingFileName(hostname string) string {
	return strings.NewReplacer("://", "_", "/", "_", ":", "_").Replace(config.NormalizeHostname(hostname)) + ".json"
}

func configDirectory() (string, error) {
	manifest := config.Active().CLI
	if value := os.Getenv(manifest.ConfigDirEnv); value != "" {
		return filepath.Join(value, manifest.ConfigDir), nil
	}
	if value := os.Getenv("XDG_CONFIG_HOME"); value != "" {
		return filepath.Join(value, manifest.ConfigDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", manifest.ConfigDir), nil
}
