package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lathe-cli/lathe/pkg/runtime"
)

const keysPath = "/api/v1/keys"

// keyRecord is the subset of the console key record the agent needs.
type keyRecord struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// keyClient is the console key API surface used to bind an agent key.
type keyClient interface {
	List(ctx context.Context, hostname string, options runtime.ClientOptions) ([]keyRecord, error)
	Create(ctx context.Context, hostname, name string, options runtime.ClientOptions) (keyRecord, string, error)
	Reveal(ctx context.Context, hostname, id string, options runtime.ClientOptions) (string, error)
	Revoke(ctx context.Context, hostname, id string, options runtime.ClientOptions) error
}

type consoleKeys struct{}

// List returns the organization keys visible to the caller. The server already
// filters revoked keys; the status check keeps the contract explicit here.
func (consoleKeys) List(ctx context.Context, hostname string, options runtime.ClientOptions) ([]keyRecord, error) {
	result, err := runtime.DoRawFull(ctx, hostname, http.MethodGet, keysPath, nil, options)
	if err != nil {
		return nil, fmt.Errorf("list Tokener API keys: %w", err)
	}
	var response struct {
		Keys []keyRecord `json:"keys"`
	}
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return nil, fmt.Errorf("decode Tokener API key list: %w", err)
	}
	return response.Keys, nil
}

// Create issues a new key and returns its record together with the plaintext.
func (consoleKeys) Create(ctx context.Context, hostname, name string, options runtime.ClientOptions) (keyRecord, string, error) {
	result, err := runtime.DoRawFull(ctx, hostname, http.MethodPost, keysPath, map[string]string{"name": name}, options)
	if err != nil {
		return keyRecord{}, "", fmt.Errorf("create Tokener agent key: %w", err)
	}
	var response struct {
		Key    string    `json:"key"`
		Record keyRecord `json:"record"`
	}
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return keyRecord{}, "", fmt.Errorf("decode Tokener agent key response: %w", err)
	}
	if strings.TrimSpace(response.Key) == "" {
		return keyRecord{}, "", errors.New("Tokener agent key response did not include a key")
	}
	return response.Record, response.Key, nil
}

// Reveal returns the plaintext of a key that has not been revoked.
func (consoleKeys) Reveal(ctx context.Context, hostname, id string, options runtime.ClientOptions) (string, error) {
	result, err := runtime.DoRawFull(ctx, hostname, http.MethodPost, keyPath(id, "reveal"), nil, options)
	if err != nil {
		return "", fmt.Errorf("reveal Tokener agent key: %w", err)
	}
	var response struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return "", fmt.Errorf("decode Tokener agent key secret: %w", err)
	}
	if strings.TrimSpace(response.Key) == "" {
		return "", errors.New("Tokener agent key reveal did not include a key")
	}
	return response.Key, nil
}

// Revoke permanently revokes a key, which also frees its name for reuse.
func (consoleKeys) Revoke(ctx context.Context, hostname, id string, options runtime.ClientOptions) error {
	if _, err := runtime.DoRawFull(ctx, hostname, http.MethodPost, keyPath(id, "revoke"), nil, options); err != nil {
		return fmt.Errorf("revoke Tokener agent key: %w", err)
	}
	return nil
}

func keyPath(id, action string) string {
	return keysPath + "/" + url.PathEscape(id) + "/" + action
}

// isDuplicateKeyName reports whether the server rejected a create because the
// name is already taken by a non-revoked key in the organization.
func isDuplicateKeyName(err error) bool {
	return hasAPIErrorCode(err, http.StatusBadRequest, "duplicate_key_name")
}

// isKeyAlreadyGone reports whether a revoke failed because the key was already
// revoked or no longer exists. Either way the caller's goal is met: the key is
// dead and its name is free.
func isKeyAlreadyGone(err error) bool {
	return hasAPIErrorCode(err, http.StatusBadRequest, "api_key_revoked") ||
		hasAPIErrorCode(err, http.StatusBadRequest, "api_key_not_found")
}

func hasAPIErrorCode(err error, status int, code string) bool {
	var requestError *runtime.HTTPError
	if !errors.As(err, &requestError) || requestError.Status != status {
		return false
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(requestError.Body, &body); err != nil {
		return false
	}
	return body.Error == code
}

// findKeyByName returns the non-revoked key carrying the given name.
func findKeyByName(records []keyRecord, name string) (keyRecord, bool) {
	return findKey(records, func(record keyRecord) bool { return record.Name == name })
}

// findKeyByNames returns the non-revoked key carrying the first of these names
// that exists, skipping empty ones. Every candidate is an exact name, so a
// caller looking for its own key can never select another machine's.
func findKeyByNames(records []keyRecord, names ...string) (keyRecord, bool) {
	for _, name := range names {
		if name == "" {
			continue
		}
		if record, found := findKeyByName(records, name); found {
			return record, true
		}
	}
	return keyRecord{}, false
}

func findKey(records []keyRecord, match func(keyRecord) bool) (keyRecord, bool) {
	for _, record := range records {
		if record.Status != "revoked" && match(record) {
			return record, true
		}
	}
	return keyRecord{}, false
}
