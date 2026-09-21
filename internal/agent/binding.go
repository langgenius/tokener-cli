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

type bindingDocument struct {
	Key   string `json:"key"`
	KeyID string `json:"keyId,omitempty"`
}

func (binding fileBinding) Load(hostname string) (bindingDocument, bool, error) {
	path, err := bindingPathFor(hostname)
	if err != nil {
		return bindingDocument{}, false, err
	}
	document, exists, err := loadBindingFile(path)
	if err != nil || exists {
		return document, exists, err
	}
	if config.NormalizeHostname(hostname) != defaultManagementHostname {
		return bindingDocument{}, false, nil
	}
	return loadBindingFile(filepath.Join(filepath.Dir(filepath.Dir(path)), "agent-key.json"))
}

func (binding fileBinding) Save(hostname string, document bindingDocument) error {
	if document.Key == "" {
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
	data, err := json.Marshal(document)
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

func loadBindingFile(path string) (bindingDocument, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return bindingDocument{}, false, nil
	}
	if err != nil {
		return bindingDocument{}, false, fmt.Errorf("read agent key binding: %w", err)
	}
	var document bindingDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return bindingDocument{}, false, fmt.Errorf("parse agent key binding: %w", err)
	}
	if strings.TrimSpace(document.Key) == "" {
		return bindingDocument{}, false, errors.New("agent key binding is empty")
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
