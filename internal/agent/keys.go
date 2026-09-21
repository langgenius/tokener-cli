package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lathe-cli/lathe/pkg/runtime"
	"github.com/spf13/cobra"
)

const (
	keysPath      = "/api/v1/keys"
	keyNamePrefix = "tokener-agent-"
)

type createdKey struct {
	ID  string
	Key string
}

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
		{"regenerate", "Revoke the bound agent key and bind a new one", func(cmd *cobra.Command) error {
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
	document, exists, err := deps.bindings.Load(hostname)
	if err != nil {
		return err
	}
	if !exists {
		_, err = fmt.Fprintf(deps.stdout, "bound: false\nhost: %s\ngateway: %s\n", hostname, gateway)
		return err
	}
	_, err = fmt.Fprintf(deps.stdout, "bound: true\nprefix: %s\nhost: %s\ngateway: %s\n", document.Key[:min(8, len(document.Key))], hostname, gateway)
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
	if replace {
		if err := revokeBoundKey(cmd.Context(), deps, target); err != nil {
			return err
		}
	}
	return createAndBind(cmd.Context(), deps, target)
}

func revokeBoundKey(ctx context.Context, deps dependencies, target agentTarget) error {
	document, exists, err := deps.bindings.Load(target.Hostname)
	if err != nil || !exists {
		return nil
	}
	if document.KeyID == "" {
		_, err = fmt.Fprintln(deps.stderr, "The previous agent key has no recorded id and is still active; revoke it with `tokener keys revoke <key-id>`.")
		return err
	}
	if err := deps.revokeKey(ctx, target.Hostname, document.KeyID, target.Options); err != nil && !isKeyAlreadyGone(err) {
		return err
	}
	return nil
}

func createAndBind(ctx context.Context, deps dependencies, target agentTarget) error {
	created, err := deps.createKey(ctx, target.Hostname, target.Options)
	if err != nil {
		return err
	}
	if err := deps.bindings.Save(target.Hostname, bindingDocument{Key: created.Key, KeyID: created.ID}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(deps.stdout, "Tokener agent key created and bound.")
	return err
}

func createKeyRequest(ctx context.Context, hostname string, options runtime.ClientOptions) (createdKey, error) {
	result, err := runtime.DoRawFull(ctx, hostname, http.MethodPost, keysPath, map[string]string{"name": randomKeyName()}, options)
	if err != nil {
		return createdKey{}, fmt.Errorf("create Tokener agent key: %w", err)
	}
	var response struct {
		Key    string `json:"key"`
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return createdKey{}, fmt.Errorf("decode Tokener agent key response: %w", err)
	}
	if strings.TrimSpace(response.Key) == "" {
		return createdKey{}, errors.New("Tokener agent key response did not include a key")
	}
	return createdKey{ID: response.Record.ID, Key: response.Key}, nil
}

func revokeKeyRequest(ctx context.Context, hostname, id string, options runtime.ClientOptions) error {
	if _, err := runtime.DoRawFull(ctx, hostname, http.MethodPost, keysPath+"/"+url.PathEscape(id)+"/revoke", nil, options); err != nil {
		return fmt.Errorf("revoke Tokener agent key: %w", err)
	}
	return nil
}

func randomKeyName() string {
	var suffix [2]byte
	_, _ = rand.Read(suffix[:])
	return keyNamePrefix + hex.EncodeToString(suffix[:])
}

func isKeyAlreadyGone(err error) bool {
	return isKeyRequestError(err, "api_key_revoked") || isKeyRequestError(err, "api_key_not_found")
}

func isKeyRequestError(err error, code string) bool {
	var requestError *runtime.HTTPError
	if !errors.As(err, &requestError) || requestError.Status != http.StatusBadRequest {
		return false
	}
	var body struct {
		Error string `json:"error"`
	}
	return json.Unmarshal(requestError.Body, &body) == nil && body.Error == code
}
