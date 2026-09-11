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
