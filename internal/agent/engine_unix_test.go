//go:build !windows

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplicitOverrideRunsHostHandshake(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rx")
	capabilities := `{"protocol":{"major":1,"minor":0},"version":"test","harnesses":["claude","codex","opencode","pi","dsh","kimi"]}`
	script := fmt.Sprintf("#!/bin/sh\nif [ -n \"${%s+x}\" ]; then exit 9; fi\nprintf '%%s\\n' '%s'\n", requestEnvironment, capabilities)
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	engine := testEngine([]byte("unused"), t.TempDir())
	engine.lookupEnv = func(name string) (string, bool) {
		return path, name == "TOKENER_RX"
	}
	resolved, err := engine.Resolve(context.Background(), "kimi")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != path {
		t.Fatalf("resolved path = %q", resolved)
	}
}

func TestExplicitOverrideRequiresCompatibleHarness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rx")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"protocol\":{\"major\":1,\"minor\":0},\"version\":\"test\",\"harnesses\":[\"claude\"]}'\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	engine := testEngine([]byte("unused"), t.TempDir())
	engine.lookupEnv = func(name string) (string, bool) {
		return path, name == "TOKENER_RX"
	}
	_, err := engine.Resolve(context.Background(), "codex")
	if err == nil || !strings.Contains(err.Error(), "does not support codex") {
		t.Fatalf("error = %v", err)
	}
}
