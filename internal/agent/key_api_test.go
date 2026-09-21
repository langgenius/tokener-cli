package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/lathe-cli/lathe/pkg/runtime"
)

var generatedKeyName = regexp.MustCompile(`^tokener-agent-[0-9a-f]{4}$`)

func TestCreateKeyRequestUsesExistingKeyCreateContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/v1/keys" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer management-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !generatedKeyName.MatchString(body["name"]) || len(body) != 1 {
			t.Fatalf("body = %v", body)
		}
		response.WriteHeader(http.StatusAccepted)
		_, _ = response.Write([]byte(`{"key":"agent-key","record":{"id":"key-id"}}`))
	}))
	defer server.Close()

	created, err := createKeyRequest(context.Background(), server.URL, runtime.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer management-token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created != (createdKey{ID: "key-id", Key: "agent-key"}) {
		t.Fatalf("created = %#v", created)
	}
}

func TestCreateKeyRequestRequiresPlaintextKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusAccepted)
		_, _ = response.Write([]byte(`{"id":"key-id"}`))
	}))
	defer server.Close()

	if _, err := createKeyRequest(context.Background(), server.URL, runtime.ClientOptions{}); err == nil {
		t.Fatal("response without plaintext key was accepted")
	}
}

func TestRevokeKeyRequestPostsToKeyRevokePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.EscapedPath() != "/api/v1/keys/key%2Fid/revoke" {
			t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
		}
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(`{}`))
	}))
	defer server.Close()

	if err := revokeKeyRequest(context.Background(), server.URL, "key/id", runtime.ClientOptions{}); err != nil {
		t.Fatal(err)
	}
}
