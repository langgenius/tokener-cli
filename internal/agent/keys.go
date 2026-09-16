package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/lathe-cli/lathe/pkg/runtime"
	"github.com/spf13/cobra"
)

func newKeyCommand(deps dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage the local Tokener agent key binding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	for _, action := range []struct {
		name, description string
		run               func(*cobra.Command) error
	}{
		{"login", "Create and bind an agent key when none is configured", func(cmd *cobra.Command) error {
			return bindKey(cmd, deps, false)
		}},
		{"regenerate", "Create and bind a new agent key", func(cmd *cobra.Command) error {
			return bindKey(cmd, deps, true)
		}},
		{"status", "Show the local Tokener agent key binding", func(cmd *cobra.Command) error {
			return statusKey(cmd, deps)
		}},
	} {
		cmd.AddCommand(&cobra.Command{
			Use:   action.name,
			Short: action.description,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return visibleError(action.run(cmd))
			},
		})
	}
	return cmd
}

func statusKey(cmd *cobra.Command, deps dependencies) error {
	hostname, _, err := deps.resolveHostname(cmd)
	if err != nil {
		return err
	}
	gateway, err := gatewayEndpointFor(hostname)
	if err != nil {
		return err
	}
	key, exists, err := deps.bindings.Load(hostname)
	if err != nil {
		return err
	}
	if !exists {
		_, err = fmt.Fprintf(deps.stdout, "bound: false\nhost: %s\ngateway: %s\n", hostname, gateway)
		return err
	}
	_, err = fmt.Fprintf(deps.stdout, "bound: true\nprefix: %s\nhost: %s\ngateway: %s\n", key[:min(8, len(key))], hostname, gateway)
	return err
}

func bindKey(cmd *cobra.Command, deps dependencies, replace bool) error {
	if _, err := deps.engine.Resolve(cmd.Context(), ""); err != nil {
		return err
	}
	target, err := deps.resolveTarget(cmd)
	if err != nil {
		return err
	}
	noticeCurrentHost(deps.stderr, target.Hostname, target.Ambiguous)
	if !replace {
		if _, exists, err := deps.bindings.Load(target.Hostname); err != nil {
			return err
		} else if exists {
			_, err = fmt.Fprintln(deps.stdout, "Tokener agent key is already bound.")
			return err
		}
	}
	return createAndBind(cmd.Context(), deps, target)
}

func createAndBind(ctx context.Context, deps dependencies, target agentTarget) error {
	key, err := deps.createKey(ctx, target.Hostname, target.Options)
	if err != nil {
		return err
	}
	if err := deps.bindings.Save(target.Hostname, key); err != nil {
		return err
	}
	_, err = fmt.Fprintln(deps.stdout, "Tokener agent key created and bound.")
	return err
}

func createKeyRequest(ctx context.Context, hostname string, options runtime.ClientOptions) (string, error) {
	result, err := runtime.DoRawFull(ctx, hostname, http.MethodPost, "/api/v1/keys", map[string]string{"name": "Tokener Agent CLI"}, options)
	if err != nil {
		return "", fmt.Errorf("create Tokener agent key: %w", err)
	}
	var response struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return "", fmt.Errorf("decode Tokener agent key response: %w", err)
	}
	if strings.TrimSpace(response.Key) == "" {
		return "", errors.New("Tokener agent key response did not include a key")
	}
	return response.Key, nil
}
