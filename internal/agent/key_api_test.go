package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lathe-cli/lathe/pkg/runtime"
)

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
		if body["name"] != "Tokener Agent CLI" || len(body) != 1 {
			t.Fatalf("body = %v", body)
		}
		response.WriteHeader(http.StatusAccepted)
		_, _ = response.Write([]byte(`{"key":"agent-key"}`))
	}))
	defer server.Close()

	key, err := createKeyRequest(context.Background(), server.URL, runtime.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer management-token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if key != "agent-key" {
		t.Fatalf("key = %q", key)
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
