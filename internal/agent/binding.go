package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lathe-cli/lathe/pkg/config"
)

type fileBinding struct {
	path func() (string, error)
}

type bindingDocument struct {
	Key string `json:"key"`
}

func newFileBinding() fileBinding {
	return fileBinding{path: bindingPath}
}

func (binding fileBinding) Load() (string, bool, error) {
	path, err := binding.path()
	if err != nil {
		return "", false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read agent key binding: %w", err)
	}
	var document bindingDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return "", false, fmt.Errorf("parse agent key binding: %w", err)
	}
	if strings.TrimSpace(document.Key) == "" {
		return "", false, errors.New("agent key binding is empty")
	}
	return document.Key, true, nil
}

func (binding fileBinding) Save(key string) error {
	if key == "" {
		return errors.New("agent key is empty")
	}
	path, err := binding.path()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create agent config directory: %w", err)
	}
	if err := restrictDirectory(dir); err != nil {
		return err
	}
	data, err := json.Marshal(bindingDocument{Key: key})
	if err != nil {
		return fmt.Errorf("encode agent key binding: %w", err)
	}
	temporary, err := os.CreateTemp(dir, ".agent-key-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary agent key binding: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict temporary agent key binding: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary agent key binding: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary agent key binding: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary agent key binding: %w", err)
	}
	if err := replaceFile(temporaryPath, path); err != nil {
		return fmt.Errorf("replace agent key binding: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict agent key binding: %w", err)
	}
	return nil
}

func bindingPath() (string, error) {
	dir, err := configDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "agent-key.json"), nil
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
