package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/lathe-cli/lathe/pkg/config"
	"github.com/lathe-cli/lathe/pkg/runtime"
)

const managementHostname = "console.tokener.dev"

func createAgentKey(ctx context.Context) (string, error) {
	hosts, err := config.LoadHosts()
	if err != nil {
		return "", fmt.Errorf("load Tokener management identity: %w", err)
	}
	entry, exists := hosts.Get(managementHostname)
	if !exists || !hasCredential(entry) {
		return "", errors.New("Tokener management login is required; run `tokener auth login --with-token`")
	}
	auth, err := runtime.NewAuthFromHost(entry)
	if err != nil {
		return "", fmt.Errorf("load Tokener management identity: %w", err)
	}
	return createKeyRequest(ctx, managementHostname, runtime.ClientOptions{Auth: auth, Insecure: entry.Insecure})
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

func hasCredential(entry config.HostEntry) bool {
	switch entry.AuthType {
	case "", "bearer":
		return entry.OAuthToken != ""
	case "apikey":
		return entry.APIKey != ""
	case "basic":
		return entry.BasicUser != ""
	default:
		return false
	}
}
