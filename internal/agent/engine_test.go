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
)

func testEngine(data []byte, root string) embeddedEngine {
	return embeddedEngine{
		data:       data,
		digest:     sha256Hex(data),
		version:    "test",
		targetOS:   runtime.GOOS,
		targetArch: runtime.GOARCH,
		cacheRoot:  func() (string, error) { return root, nil },
		lookupEnv:  func(string) (string, bool) { return "", false },
	}
}

func TestEmbeddedEngineExtractionIsRepeatableAndRepairsCorruption(t *testing.T) {
	root := t.TempDir()
	data := []byte("rx-engine")
	engine := testEngine(data, root)

	first, err := engine.Resolve(context.Background(), "codex")
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.Resolve(context.Background(), "codex")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("cache paths = %q/%q", first, second)
	}
	if body, err := os.ReadFile(first); err != nil || !slices.Equal(body, data) {
		t.Fatalf("cached body/error = %q/%v", body, err)
	}
	if err := os.WriteFile(first, []byte("corrupt"), 0o700); err != nil {
		t.Fatal(err)
	}
	repaired, err := engine.Resolve(context.Background(), "codex")
	if err != nil {
		t.Fatal(err)
	}
	if repaired != first {
		t.Fatalf("repaired path = %q", repaired)
	}
	if body, err := os.ReadFile(first); err != nil || !slices.Equal(body, data) {
		t.Fatalf("repaired body/error = %q/%v", body, err)
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

func TestEmbeddedEngineKeepsOldDigestForRollback(t *testing.T) {
	root := t.TempDir()
	oldEngine := testEngine([]byte("old-rx"), root)
	newEngine := testEngine([]byte("new-rx"), root)
	oldPath, err := oldEngine.Resolve(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	newPath, err := newEngine.Resolve(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if oldPath == newPath {
		t.Fatalf("upgrade reused %q", oldPath)
	}
	for _, path := range []string{oldPath, newPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("cached engine %q: %v", path, err)
		}
	}
	rollbackPath, err := oldEngine.Resolve(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if rollbackPath != oldPath {
		t.Fatalf("rollback path = %q, expected %q", rollbackPath, oldPath)
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
			Endpoint:      gatewayEndpoint,
			CredentialEnv: credentialEnv,
		},
		StateDir:         filepath.Join(t.TempDir(), "state"),
		PermissionPolicy: "standard",
		InstallPolicy:    "prompt",
	}
	args, environment, err := launchSpec(
		request,
		[]string{"--resume", "session-1"},
		"gateway-secret",
		[]string{"PATH=/bin", requestEnvironment + "=old", credentialEnv + "=old"},
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
	for _, entry := range environment {
		name, value, _ := strings.Cut(entry, "=")
		switch name {
		case requestEnvironment:
			payload = value
		case credentialEnv:
			if value != "gateway-secret" {
				t.Fatalf("credential environment = %q", value)
			}
		}
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
