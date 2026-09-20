package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/langgenius/tokener-cli/internal/rxsnapshot"
)

func testEngine(data []byte, root string) embeddedEngine {
	return embeddedEngine{
		data:       data,
		digest:     rxsnapshot.Digest(data),
		version:    "test",
		targetOS:   runtime.GOOS,
		targetArch: runtime.GOARCH,
		cacheRoot:  func() (string, error) { return root, nil },
		lookupEnv:  func(string) (string, bool) { return "", false },
	}
}

func TestEmbeddedEngineCacheLifecycle(t *testing.T) {
	root := t.TempDir()
	engine := testEngine([]byte("rx-engine"), root)
	var cached string
	for _, step := range []string{"extract", "reuse", "repair", "rollback"} {
		if step == "repair" {
			if err := os.WriteFile(cached, []byte("corrupt"), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		if step == "rollback" {
			newer := testEngine([]byte("new-rx"), root)
			path, err := newer.Resolve(t.Context(), "")
			if err != nil || path == cached {
				t.Fatalf("new engine path = %q, error = %v", path, err)
			}
			for _, retained := range []string{cached, path} {
				if _, err := os.Stat(retained); err != nil {
					t.Fatal(err)
				}
			}
		}
		path, err := engine.Resolve(t.Context(), "codex")
		if err != nil {
			t.Fatal(err)
		}
		if cached != "" && path != cached {
			t.Fatalf("cache path = %q, expected %q", path, cached)
		}
		cached = path
		body, err := os.ReadFile(path)
		if err != nil || !slices.Equal(body, engine.data) {
			t.Fatalf("cached body = %q, error = %v", body, err)
		}
	}
}

func TestEmbeddedEngineConcurrentExtractionUsesOneDigestPath(t *testing.T) {
	root := t.TempDir()
	engine := testEngine([]byte("concurrent-rx-engine"), root)
	paths := make(chan string, 16)
	errors := make(chan error, 16)
	var group sync.WaitGroup
	for range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			path, err := engine.Resolve(context.Background(), "")
			paths <- path
			errors <- err
		}()
	}
	group.Wait()
	close(paths)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var expected string
	for path := range paths {
		if expected == "" {
			expected = path
		}
		if path != expected {
			t.Fatalf("cache path = %q, expected %q", path, expected)
		}
	}
}

func TestEmbeddedEngineRejectsExplicitEmptyOverride(t *testing.T) {
	engine := testEngine([]byte("rx"), t.TempDir())
	engine.lookupEnv = func(name string) (string, bool) {
		return "", name == "TOKENER_RX"
	}
	_, err := engine.Resolve(context.Background(), "")
	if err == nil || err.Error() != "TOKENER_RX is empty" {
		t.Fatalf("error = %v", err)
	}
}

func TestLaunchSpecKeepsKeyOutOfRequestAndArguments(t *testing.T) {
	request := hostRequest{
		Gateway: gatewayProfile{
			ProviderID:    "tokener",
			Name:          "Tokener",
			Endpoint:      defaultGatewayEndpoint,
			CredentialEnv: credentialEnv,
		},
		StateDir:      filepath.Join(t.TempDir(), "state"),
		InstallPolicy: "prompt",
	}
	args, environment, err := launchSpec(
		request,
		[]string{"--resume", "session-1"},
		"gateway-secret",
		[]string{"PATH=/bin", requestEnvironment + "=old", credentialEnv + "=old", claudeExperimentalBetas + "=0"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(args, []string{"host", "--", "--resume", "session-1"}) {
		t.Fatalf("args = %v", args)
	}
	if strings.Contains(strings.Join(args, "\x00"), "gateway-secret") {
		t.Fatalf("key leaked into args: %v", args)
	}
	var payload string
	var sawBetas bool
	for _, entry := range environment {
		name, value, _ := strings.Cut(entry, "=")
		switch name {
		case requestEnvironment:
			payload = value
		case credentialEnv:
			if value != "gateway-secret" {
				t.Fatalf("credential environment = %q", value)
			}
		case claudeExperimentalBetas:
			sawBetas = true
			if value != "1" {
				t.Fatalf("claude experimental betas = %q", value)
			}
		}
	}
	if !sawBetas {
		t.Fatal("missing CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS")
	}
	if payload == "" || strings.Contains(payload, "gateway-secret") {
		t.Fatalf("request payload = %q", payload)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatal(err)
	}
	if _, exists := decoded["harness"]; exists {
		t.Fatalf("empty harness was serialized: %v", decoded)
	}
}
