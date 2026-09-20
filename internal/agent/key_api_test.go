package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/lathe-cli/lathe/pkg/runtime"
)

func managementOptions() runtime.ClientOptions {
	return runtime.ClientOptions{Headers: map[string]string{"Authorization": "Bearer management-token"}}
}

func TestConsoleKeysCreateUsesExistingKeyCreateContract(t *testing.T) {
	name := "Tokener Agent CLI · box-abc123"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != keysPath {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer management-token" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["name"] != name || len(body) != 1 {
			t.Errorf("body = %v", body)
		}
		response.WriteHeader(http.StatusAccepted)
		_, _ = response.Write([]byte(`{"key":"agent-key","record":{"id":"key-1","name":"` + name + `","status":"active","prefix":"agent-ke"}}`))
	}))
	defer server.Close()

	client := consoleKeys{}
	record, key, err := client.Create(context.Background(), server.URL, name, managementOptions())
	if err != nil {
		t.Fatal(err)
	}
	if key != "agent-key" {
		t.Fatalf("key = %q", key)
	}
	if record.ID != "key-1" || record.Name != name {
		t.Fatalf("record = %#v", record)
	}
}

func TestConsoleKeysCreateRequiresPlaintextKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusAccepted)
		_, _ = response.Write([]byte(`{"record":{"id":"key-id"}}`))
	}))
	defer server.Close()

	client := consoleKeys{}
	if _, _, err := client.Create(context.Background(), server.URL, "name", runtime.ClientOptions{}); err == nil {
		t.Fatal("response without plaintext key was accepted")
	}
}

func TestConsoleKeysListRevealAndRevokeUseTheDocumentedPaths(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		seen = append(seen, request.Method+" "+request.URL.EscapedPath())
		switch request.URL.EscapedPath() {
		case keysPath:
			_, _ = response.Write([]byte(`{"connected":true,"keys":[{"id":"key-1","name":"first","status":"active","prefix":"sk-aaaaa"}]}`))
		case keysPath + "/key%201/reveal":
			_, _ = response.Write([]byte(`{"key":"sk-revealed"}`))
		case keysPath + "/key%201/revoke":
			_, _ = response.Write([]byte(`{"record":{"id":"key 1","status":"revoked"}}`))
		default:
			t.Errorf("unexpected path %q", request.URL.EscapedPath())
		}
	}))
	defer server.Close()

	client := consoleKeys{}
	records, err := client.List(context.Background(), server.URL, managementOptions())
	if err != nil {
		t.Fatal(err)
	}
	// Fields the agent does not use, such as prefix, are ignored rather than
	// failing the decode.
	if len(records) != 1 || records[0].ID != "key-1" || records[0].Name != "first" {
		t.Fatalf("records = %#v", records)
	}
	// The identifier is escaped into the path rather than interpolated raw.
	secret, err := client.Reveal(context.Background(), server.URL, "key 1", managementOptions())
	if err != nil || secret != "sk-revealed" {
		t.Fatalf("reveal = %q/%v", secret, err)
	}
	if err := client.Revoke(context.Background(), server.URL, "key 1", managementOptions()); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET " + keysPath,
		"POST " + keysPath + "/key%201/reveal",
		"POST " + keysPath + "/key%201/revoke",
	}
	if !slices.Equal(seen, want) {
		t.Fatalf("requests = %q, want %q", seen, want)
	}
}

func TestIsDuplicateKeyNameMatchesOnlyTheServersBareCode(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{"duplicate", duplicateKeyNameResponse(), true},
		{"wrapped duplicate", fmt.Errorf("create Tokener agent key: %w", duplicateKeyNameResponse()), true},
		{"other 400", badRequest(`{"error":"invalid_key_name"}`), false},
		{"same code on another status", &runtime.HTTPError{Status: 500, Body: []byte(`{"error":"duplicate_key_name"}`)}, false},
		{"unparsable body", badRequest("not json"), false},
		{"unrelated error", errors.New("duplicate_key_name"), false},
		{"no error", nil, false},
	} {
		if got := isDuplicateKeyName(test.err); got != test.want {
			t.Fatalf("%s: isDuplicateKeyName = %t", test.name, got)
		}
	}
}

func badRequest(body string) error {
	return &runtime.HTTPError{Status: http.StatusBadRequest, Body: []byte(body)}
}

func TestFindKeyIgnoresRevokedRecords(t *testing.T) {
	records := []keyRecord{
		{ID: "old", Name: "shared", Status: "revoked"},
		{ID: "live", Name: "shared", Status: "disabled"},
		{ID: "legacy", Name: agentKeyNamePrefix, Status: "active"},
	}
	// A revoked key no longer holds its name, and a disabled one still does.
	if record, found := findKeyByName(records, "shared"); !found || record.ID != "live" {
		t.Fatalf("by name = %#v/%t", record, found)
	}
	if _, found := findKeyByName(records, "absent"); found {
		t.Fatal("absent name matched")
	}
	// Empty candidates are skipped, and the earliest name that exists wins.
	if record, found := findKeyByNames(records, "", "absent", agentKeyNamePrefix); !found || record.ID != "legacy" {
		t.Fatalf("by names = %#v/%t", record, found)
	}
	if record, found := findKeyByNames(records, "shared", agentKeyNamePrefix); !found || record.ID != "live" {
		t.Fatalf("earlier name did not win = %#v/%t", record, found)
	}
	if _, found := findKeyByNames(records, "", "absent"); found {
		t.Fatal("absent names matched")
	}
}
