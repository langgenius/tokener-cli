package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// consoleStub is a minimal stand-in for the console key API that enforces the
// rule behind issue #37: a name is unique per organization across keys that are
// not revoked. It exists so the real HTTP client, not a fake, is exercised.
type consoleStub struct {
	records []keyRecord
	secrets map[string]string
	calls   []string
}

func newConsoleStub(t *testing.T) (*consoleStub, string) {
	t.Helper()
	stub := &consoleStub{secrets: map[string]string{}}
	server := httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(server.Close)
	return stub, server.URL
}

func (stub *consoleStub) serve(response http.ResponseWriter, request *http.Request) {
	path := request.URL.EscapedPath()
	stub.calls = append(stub.calls, request.Method+" "+path)
	response.Header().Set("Content-Type", "application/json")
	switch {
	case request.Method == http.MethodGet && path == keysPath:
		visible := []keyRecord{}
		for _, record := range stub.records {
			if record.Status != "revoked" {
				visible = append(visible, record)
			}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"connected": true, "keys": visible})
	case request.Method == http.MethodPost && path == keysPath:
		stub.create(response, request)
	case strings.HasSuffix(path, "/reveal"):
		stub.reveal(response, keyIDFromPath(path, "reveal"))
	case strings.HasSuffix(path, "/revoke"):
		stub.revoke(response, keyIDFromPath(path, "revoke"))
	default:
		response.WriteHeader(http.StatusNotFound)
	}
}

func (stub *consoleStub) create(response http.ResponseWriter, request *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(request.Body).Decode(&body)
	if len([]rune(body.Name)) > maxKeyNameLength {
		response.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(response).Encode(map[string]string{"error": "invalid_key_name"})
		return
	}
	if _, taken := findKeyByName(stub.records, body.Name); taken {
		response.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(response).Encode(map[string]string{"error": "duplicate_key_name"})
		return
	}
	secret := fmt.Sprintf("sk-%08d-secret", len(stub.records)+1)
	record := keyRecord{
		ID:     fmt.Sprintf("key-%d", len(stub.records)+1),
		Name:   body.Name,
		Status: "active",
	}
	stub.records = append(stub.records, record)
	stub.secrets[record.ID] = secret
	response.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(response).Encode(map[string]any{"key": secret, "record": record})
}

func (stub *consoleStub) reveal(response http.ResponseWriter, id string) {
	for _, record := range stub.records {
		if record.ID != id {
			continue
		}
		if record.Status == "revoked" {
			response.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(response).Encode(map[string]string{"error": "api_key_revoked"})
			return
		}
		_ = json.NewEncoder(response).Encode(map[string]string{"key": stub.secrets[id]})
		return
	}
	response.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(response).Encode(map[string]string{"error": "api_key_not_found"})
}

func (stub *consoleStub) revoke(response http.ResponseWriter, id string) {
	for index, record := range stub.records {
		if record.ID != id {
			continue
		}
		if record.Status == "revoked" {
			response.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(response).Encode(map[string]string{"error": "api_key_revoked"})
			return
		}
		stub.records[index].Status = "revoked"
		_ = json.NewEncoder(response).Encode(map[string]any{"record": stub.records[index]})
		return
	}
	response.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(response).Encode(map[string]string{"error": "api_key_not_found"})
}

func keyIDFromPath(path, action string) string {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(path, keysPath+"/"), "/"+action)
	unescaped, err := url.PathUnescape(trimmed)
	if err != nil {
		return trimmed
	}
	return unescaped
}

// TestTwoMachinesBindThroughTheRealClient walks the whole flow over HTTP: two
// machines log in against one organization, each rerunning login, and one then
// rotates. It fails on the pre-fix behaviour, where the shared name made the
// second machine's create return duplicate_key_name.
func TestTwoMachinesBindThroughTheRealClient(t *testing.T) {
	stub, endpoint := newConsoleStub(t)
	machines := []machineIdentity{
		{Hostname: "macbook", Digest: "aaa111"},
		{Hostname: "macbook", Digest: "bbb222"},
	}

	bindings := map[string]agentBinding{}
	for _, identity := range machines {
		for run := range 2 {
			fixture := newAgentFixture(t)
			fixture.dependencies.keys = consoleKeys{}
			fixture.dependencies.identity = func() (machineIdentity, error) { return identity, nil }
			fixture.dependencies.resolveTarget = func(*cobra.Command) (agentTarget, error) {
				return agentTarget{Hostname: endpoint}, nil
			}
			fixture.binding.exists = false
			if err := fixture.execute("key", "login"); err != nil {
				t.Fatalf("%s run %d: %v", identity.Digest, run, err)
			}
			if len(fixture.binding.saved) != 1 {
				t.Fatalf("%s run %d binding = %#v", identity.Digest, run, fixture.binding.saved)
			}
			stored := fixture.binding.saved[0]
			if previous, seen := bindings[identity.Digest]; seen && previous != stored {
				// The second login must reuse the key, not mint a new one.
				t.Fatalf("%s run %d rebound %#v over %#v", identity.Digest, run, stored, previous)
			}
			bindings[identity.Digest] = stored
		}
	}

	if len(stub.records) != 2 {
		t.Fatalf("organization holds %d keys: %#v", len(stub.records), stub.records)
	}
	if bindings["aaa111"].Key == bindings["bbb222"].Key {
		t.Fatal("both machines bound the same secret")
	}

	// Rotation replaces only the rotating machine's key.
	rotating := bindings["aaa111"]
	fixture := newAgentFixture(t)
	fixture.dependencies.keys = consoleKeys{}
	fixture.dependencies.identity = func() (machineIdentity, error) { return machines[0], nil }
	fixture.dependencies.resolveTarget = func(*cobra.Command) (agentTarget, error) {
		return agentTarget{Hostname: endpoint}, nil
	}
	fixture.binding.binding, fixture.binding.exists = rotating, true
	if err := fixture.execute("key", "regenerate"); err != nil {
		t.Fatal(err)
	}
	rotated := fixture.binding.saved[0]
	if rotated.Key == rotating.Key || rotated.KeyID == rotating.KeyID {
		t.Fatalf("rotation reused the old key: %#v", rotated)
	}
	if rotated.Name != rotating.Name {
		t.Fatalf("rotation changed the name from %q to %q", rotating.Name, rotated.Name)
	}
	live := []string{}
	for _, record := range stub.records {
		if record.Status != "revoked" {
			live = append(live, record.Name)
		}
	}
	slices.Sort(live)
	want := []string{machines[0].keyName(1), machines[1].keyName(1)}
	slices.Sort(want)
	if !slices.Equal(live, want) {
		t.Fatalf("live keys = %q, want %q", live, want)
	}
}
