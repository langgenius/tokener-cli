package agent

import (
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
	Source    string
	Gateway   string
	Ambiguous bool
	Options   runtime.ClientOptions
}

func resolveAgentTarget(cmd *cobra.Command) (agentTarget, error) {
	hostname, source, ambiguous, err := resolveManagementHostname(cmd)
	if err != nil {
		return agentTarget{}, err
	}
	gateway, err := gatewayEndpointFor(hostname)
	if err != nil {
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
	insecure := entry.Insecure
	if value, flagErr := cmd.Root().PersistentFlags().GetBool("insecure"); flagErr == nil && value {
		insecure = true
	}
	auth, err := runtime.NewAuthFromHost(entry)
	if err != nil {
		return agentTarget{}, fmt.Errorf("load Tokener management identity: %w", err)
	}
	return agentTarget{
		Hostname:  hostname,
		Source:    source,
		Gateway:   gateway,
		Ambiguous: ambiguous,
		Options:   runtime.ClientOptions{Auth: auth, Insecure: insecure},
	}, nil
}

func resolveManagementHostname(cmd *cobra.Command) (string, string, bool, error) {
	hosts, err := config.LoadHosts()
	if err != nil {
		return "", "", false, err
	}
	if hostname, _ := cmd.Root().PersistentFlags().GetString("hostname"); hostname != "" {
		return config.NormalizeHostname(hostname), runtime.HostSourceFlag, len(hosts.Names()) > 1, nil
	}
	if hostname := os.Getenv(config.Active().CLI.HostEnv); hostname != "" {
		return config.NormalizeHostname(hostname), runtime.HostSourceEnv, len(hosts.Names()) > 1, nil
	}
	names := hosts.Names()
	ambiguous := len(names) > 1
	if selected := hosts.Selected(); selected != "" {
		return selected, runtime.HostSourceSelected, ambiguous, nil
	}
	return defaultManagementHostname, runtime.HostSourceCodegenDefault, ambiguous, nil
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
