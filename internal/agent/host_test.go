package agent

import (
	"testing"

	"github.com/lathe-cli/lathe/pkg/config"
	"github.com/lathe-cli/lathe/pkg/runtime"
	"github.com/spf13/cobra"
)

func TestGatewayEndpointForKnownHosts(t *testing.T) {
	tests := []struct {
		hostname string
		want     string
	}{
		{defaultManagementHostname, defaultGatewayEndpoint},
		{"console-staging.tokener.dev", "https://api-staging.tokener.dev/v1"},
		{"console.tokener.ai", "https://api.tokener.ai/v1"},
		{"http://localhost:3000", localGatewayEndpoint},
		{"localhost:3000", localGatewayEndpoint},
		{"http://127.0.0.1:3000", localGatewayEndpoint},
	}
	for _, tt := range tests {
		got, err := gatewayEndpointFor(tt.hostname)
		if err != nil {
			t.Fatalf("%s: %v", tt.hostname, err)
		}
		if got != tt.want {
			t.Fatalf("%s: gateway = %q, want %q", tt.hostname, got, tt.want)
		}
	}
}

func TestGatewayEndpointForRejectsUnknownHosts(t *testing.T) {
	for _, hostname := range []string{"api.tokener.dev", "example.com", "http://localhost:9000"} {
		if _, err := gatewayEndpointFor(hostname); err == nil {
			t.Fatalf("%s was accepted", hostname)
		}
	}
}

func bindHostResolutionTest(t *testing.T) *cobra.Command {
	t.Helper()
	root := t.TempDir()
	t.Setenv("TOKENER_CONFIG_DIR", root)
	t.Setenv("TOKENER_HOST", "")
	config.Bind(&config.Manifest{CLI: config.CLIInfo{
		Name:         "tokener",
		ConfigDir:    "tokener",
		ConfigDirEnv: "TOKENER_CONFIG_DIR",
		HostEnv:      "TOKENER_HOST",
	}})
	rootCmd := &cobra.Command{Use: "tokener"}
	rootCmd.PersistentFlags().String("hostname", "", "")
	rootCmd.PersistentFlags().Bool("insecure", false, "")
	child := &cobra.Command{Use: "agent"}
	rootCmd.AddCommand(child)
	return child
}

func writeTestHosts(t *testing.T, selected string, names ...string) {
	t.Helper()
	hosts, err := config.LoadHosts()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range hosts.Names() {
		hosts.Delete(name)
	}
	for _, name := range names {
		hosts.Set(name, config.HostEntry{AuthType: "bearer", OAuthToken: "token-" + name})
	}
	if selected != "" {
		hosts.Select(selected)
	}
	if err := hosts.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestResolveManagementHostnameOrder(t *testing.T) {
	child := bindHostResolutionTest(t)
	writeTestHosts(t, "console-staging.tokener.dev", "console.tokener.dev", "console-staging.tokener.dev", "http://localhost:3000")

	hostname, source, ambiguous, err := resolveManagementHostname(child)
	if err != nil {
		t.Fatal(err)
	}
	if hostname != "console-staging.tokener.dev" || source != runtime.HostSourceSelected || !ambiguous {
		t.Fatalf("selected = %q/%s/%t", hostname, source, ambiguous)
	}

	t.Setenv("TOKENER_HOST", "http://localhost:3000")
	hostname, source, _, err = resolveManagementHostname(child)
	if err != nil {
		t.Fatal(err)
	}
	if hostname != "http://localhost:3000" || source != runtime.HostSourceEnv {
		t.Fatalf("env = %q/%s", hostname, source)
	}

	if err := child.Root().PersistentFlags().Set("hostname", "console.tokener.dev"); err != nil {
		t.Fatal(err)
	}
	hostname, source, _, err = resolveManagementHostname(child)
	if err != nil {
		t.Fatal(err)
	}
	if hostname != "console.tokener.dev" || source != runtime.HostSourceFlag {
		t.Fatalf("flag = %q/%s", hostname, source)
	}

	if err := child.Root().PersistentFlags().Set("hostname", ""); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOKENER_HOST", "")
	writeTestHosts(t, "", "console.tokener.dev", "console-staging.tokener.dev")
	hostname, source, ambiguous, err = resolveManagementHostname(child)
	if err != nil {
		t.Fatal(err)
	}
	if hostname != defaultManagementHostname || source != runtime.HostSourceCodegenDefault || !ambiguous {
		t.Fatalf("default = %q/%s/%t", hostname, source, ambiguous)
	}
}

func TestResolveAgentTargetRequiresCredentials(t *testing.T) {
	child := bindHostResolutionTest(t)
	writeTestHosts(t, "console-staging.tokener.dev", "console-staging.tokener.dev")

	target, err := resolveAgentTarget(child)
	if err != nil {
		t.Fatal(err)
	}
	if target.Hostname != "console-staging.tokener.dev" || target.Gateway != "https://api-staging.tokener.dev/v1" {
		t.Fatalf("target = %#v", target)
	}

	if err := child.Root().PersistentFlags().Set("hostname", "console.tokener.dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveAgentTarget(child); err == nil {
		t.Fatal("expected missing credentials error")
	}
}
