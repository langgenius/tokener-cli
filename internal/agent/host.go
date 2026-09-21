package agent

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lathe-cli/lathe/pkg/config"
	"github.com/lathe-cli/lathe/pkg/runtime"
	"github.com/spf13/cobra"
)

const (
	defaultManagementHostname = "console.tokener.dev"
	defaultGatewayEndpoint    = "https://api.tokener.dev/v1"
	localGatewayEndpoint      = "http://localhost:8080/v1"
)

type agentTarget struct {
	Hostname  string
	Ambiguous bool
	Options   runtime.ClientOptions
}

func resolveAgentTarget(cmd *cobra.Command) (agentTarget, error) {
	hostname, ambiguous, err := resolveManagementHostname(cmd)
	if err != nil {
		return agentTarget{}, err
	}
	if _, err := gatewayEndpointFor(hostname); err != nil {
		return agentTarget{}, err
	}
	hosts, err := config.LoadHosts()
	if err != nil {
		return agentTarget{}, fmt.Errorf("load Tokener management identity: %w", err)
	}
	entry, exists := hosts.Get(hostname)
	if !exists || !hasCredential(entry) {
		return agentTarget{}, fmt.Errorf(
			"Tokener management login is required for %s; run `tokener auth login --hostname %s`",
			hostname,
			hostname,
		)
	}
	insecure, _ := cmd.Root().PersistentFlags().GetBool("insecure")
	auth, err := runtime.NewAuthFromHost(entry)
	if err != nil {
		return agentTarget{}, fmt.Errorf("load Tokener management identity: %w", err)
	}
	return agentTarget{
		Hostname:  hostname,
		Ambiguous: ambiguous,
		Options:   runtime.ClientOptions{Auth: auth, Insecure: insecure || entry.Insecure},
	}, nil
}

func resolveManagementHostname(cmd *cobra.Command) (string, bool, error) {
	hosts, err := config.LoadHosts()
	if err != nil {
		return "", false, err
	}
	hostname, _ := cmd.Root().PersistentFlags().GetString("hostname")
	hostname = cmp.Or(hostname, os.Getenv(config.Active().CLI.HostEnv), hosts.Selected(), defaultManagementHostname)
	return config.NormalizeHostname(hostname), len(hosts.Names()) > 1, nil
}

func gatewayEndpointFor(hostname string) (string, error) {
	normalized := config.NormalizeHostname(hostname)
	if normalized == "" {
		return "", fmt.Errorf("management hostname is empty")
	}
	if isLocalConsole(normalized) {
		return localGatewayEndpoint, nil
	}
	if strings.HasPrefix(normalized, "http://") {
		return "", fmt.Errorf("unsupported management hostname %q", hostname)
	}
	label, rest, ok := strings.Cut(normalized, ".")
	if !ok || rest == "" {
		return "", fmt.Errorf("unsupported management hostname %q", hostname)
	}
	if label != "console" && !strings.HasPrefix(label, "console-") {
		return "", fmt.Errorf("unsupported management hostname %q; expected a console* host", hostname)
	}
	apiLabel := "api" + strings.TrimPrefix(label, "console")
	return "https://" + apiLabel + "." + rest + "/v1", nil
}

func isLocalConsole(hostname string) bool {
	switch config.NormalizeHostname(hostname) {
	case "http://localhost:3000", "http://localhost", "localhost:3000", "localhost",
		"http://127.0.0.1:3000", "http://127.0.0.1", "127.0.0.1:3000", "127.0.0.1":
		return true
	default:
		return false
	}
}

func noticeCurrentHost(stderr io.Writer, hostname string, ambiguous bool) {
	if ambiguous && hostname != "" {
		_, _ = fmt.Fprintf(stderr, "current host: %s\n", hostname)
	}
}

func hasCredential(entry config.HostEntry) bool {
	switch entry.AuthType {
	case "", "bearer":
		return entry.OAuthToken != ""
	case "apikey":
		return entry.APIKey != ""
	case "basic":
		return entry.BasicUser != ""
	default:
		return false
	}
}

// this comment should be rejected
