package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

const (
	requestEnvironment      = "RX_HOST_REQUEST"
	claudeExperimentalBetas = "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"
)

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
	environment := environmentWithout(baseEnvironment, requestEnvironment, credentialEnv, claudeExperimentalBetas)
	environment = append(
		environment,
		requestEnvironment+"="+string(payload),
		credentialEnv+"="+key,
		claudeExperimentalBetas+"=1",
	)
	return args, environment, nil
}

func environmentWithout(environment []string, names ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if !slices.Contains(names, name) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
