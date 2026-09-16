package agent

import (
	"testing"

	"github.com/lathe-cli/lathe/pkg/config"
	"github.com/spf13/cobra"
)

func TestGatewayEndpointFor(t *testing.T) {
	for _, test := range []struct{ hostname, want string }{
		{defaultManagementHostname, defaultGatewayEndpoint},
		{"console-staging.tokener.dev", "https://api-staging.tokener.dev/v1"},
		{"console.tokener.ai", "https://api.tokener.ai/v1"},
		{"http://localhost:3000", localGatewayEndpoint},
		{"localhost:3000", localGatewayEndpoint},
		{"http://127.0.0.1:3000", localGatewayEndpoint},
		{"api.tokener.dev", ""},
		{"example.com", ""},
		{"http://localhost:9000", ""},
	} {
		got, err := gatewayEndpointFor(test.hostname)
		if got != test.want || (err != nil) != (test.want == "") {
			t.Fatalf("%s: gateway/error = %q/%v, want %q", test.hostname, got, err, test.want)
		}
	}
}

func bindHostResolutionTest(t *testing.T) *cobra.Command {
	t.Helper()
	bindAgentTestManifest(t)
	t.Setenv("TOKENER_HOST", "")
	root := &cobra.Command{Use: "tokener"}
	root.PersistentFlags().String("hostname", "", "")
	root.PersistentFlags().Bool("insecure", false, "")
	child := &cobra.Command{Use: "agent"}
	root.AddCommand(child)
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
	for _, test := range []struct{ selected, environment, flag, want string }{
		{"console-staging.tokener.dev", "", "", "console-staging.tokener.dev"},
		{"console-staging.tokener.dev", "http://localhost:3000", "", "http://localhost:3000"},
		{"console-staging.tokener.dev", "http://localhost:3000", defaultManagementHostname, defaultManagementHostname},
		{"", "", "", defaultManagementHostname},
	} {
		child := bindHostResolutionTest(t)
		writeTestHosts(t, test.selected, defaultManagementHostname, "console-staging.tokener.dev", "http://localhost:3000")
		t.Setenv("TOKENER_HOST", test.environment)
		if err := child.Root().PersistentFlags().Set("hostname", test.flag); err != nil {
			t.Fatal(err)
		}
		hostname, ambiguous, err := resolveManagementHostname(child)
		if err != nil || hostname != test.want || !ambiguous {
			t.Fatalf("%#v: host/ambiguous/error = %q/%t/%v", test, hostname, ambiguous, err)
		}
	}
}

func TestResolveAgentTargetRequiresCredentials(t *testing.T) {
	child := bindHostResolutionTest(t)
	writeTestHosts(t, "console-staging.tokener.dev", "console-staging.tokener.dev")
	target, err := resolveAgentTarget(child)
	if err != nil || target.Hostname != "console-staging.tokener.dev" {
		t.Fatalf("target/error = %#v/%v", target, err)
	}
	if err := child.Root().PersistentFlags().Set("hostname", defaultManagementHostname); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveAgentTarget(child); err == nil {
		t.Fatal("expected missing credentials error")
	}
}
