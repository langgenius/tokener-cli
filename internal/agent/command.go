package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lathe-cli/lathe/pkg/runtime"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const credentialEnv = "TOKENER_API_KEY"

var harnesses = []string{"claude", "codex", "opencode", "pi", "dsh", "kimi"}

type engineResolver interface {
	Resolve(context.Context, string) (string, error)
}

type keyBinding interface {
	Load(hostname string) (string, bool, error)
	Save(hostname, key string) error
}

type dependencies struct {
	engine          engineResolver
	bindings        keyBinding
	resolveHostname func(*cobra.Command) (string, string, bool, error)
	resolveTarget   func(*cobra.Command) (agentTarget, error)
	createKey       func(context.Context, agentTarget) (string, error)
	launch          func(string, hostRequest, []string, string) error
	interactive     func() bool
	stdin           io.Reader
	stdout          io.Writer
	stderr          io.Writer
}

func NewCommand() *cobra.Command {
	return newCommand(dependencies{
		engine:          newEmbeddedEngine(),
		bindings:        newFileBinding(),
		resolveHostname: resolveManagementHostname,
		resolveTarget:   resolveAgentTarget,
		createKey: func(ctx context.Context, target agentTarget) (string, error) {
			return createKeyRequest(ctx, target.Hostname, target.Options)
		},
		launch: launchEngine,
		interactive: func() bool {
			return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
		},
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	})
}

func newCommand(deps dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "agent [harness] [args...]",
		Short:              "Run coding agents through the Tokener Gateway",
		Long:               agentLong(),
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			if len(args) == 0 && !deps.interactive() {
				return missingHarness(cmd, deps)
			}
			return visibleError(runAgent(cmd, deps, args))
		},
	}
	cmd.SetIn(deps.stdin)
	cmd.SetOut(deps.stdout)
	cmd.SetErr(deps.stderr)
	cmd.AddCommand(newKeyCommand(deps))
	return cmd
}

func agentLong() string {
	return "Run coding agents through the Tokener Gateway.\n\nAvailable harnesses: " + strings.Join(harnesses, ", ")
}

func missingHarness(cmd *cobra.Command, deps dependencies) error {
	if err := cmd.Usage(); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(deps.stderr, "\nAvailable harnesses: %s\n", strings.Join(harnesses, ", ")); err != nil {
		return err
	}
	return runtime.NewError(
		runtime.CodeUsage,
		runtime.ExitUsage,
		"harness is required",
		"run `tokener agent --help`",
		fmt.Errorf("available harnesses: %s", strings.Join(harnesses, ", ")),
	)
}

func newKeyCommand(deps dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage the local Tokener agent key binding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "login",
		Short: "Create and bind an agent key when none is configured",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return visibleError(loginKey(cmd, deps))
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "regenerate",
		Short: "Create and bind a new agent key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return visibleError(regenerateKey(cmd, deps))
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the local Tokener agent key binding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return visibleError(statusKey(cmd, deps))
		},
	})
	return cmd
}

func statusKey(cmd *cobra.Command, deps dependencies) error {
	hostname, _, _, err := deps.resolveHostname(cmd)
	if err != nil {
		return err
	}
	gateway, err := gatewayEndpointFor(hostname)
	if err != nil {
		return err
	}
	key, exists, err := deps.bindings.Load(hostname)
	if err != nil {
		return err
	}
	if !exists {
		_, err = fmt.Fprintf(deps.stdout, "bound: false\nhost: %s\ngateway: %s\n", hostname, gateway)
		return err
	}
	_, err = fmt.Fprintf(deps.stdout, "bound: true\nprefix: %s\nhost: %s\ngateway: %s\n", keyPrefix(key), hostname, gateway)
	return err
}

func keyPrefix(key string) string {
	const visible = 8
	if len(key) <= visible {
		return key
	}
	return key[:visible]
}

func loginKey(cmd *cobra.Command, deps dependencies) error {
	if _, err := deps.engine.Resolve(cmd.Context(), ""); err != nil {
		return err
	}
	target, err := deps.resolveTarget(cmd)
	if err != nil {
		return err
	}
	noticeCurrentHost(deps.stderr, target.Hostname, target.Ambiguous)
	if _, exists, err := deps.bindings.Load(target.Hostname); err != nil {
		return err
	} else if exists {
		_, err = fmt.Fprintln(deps.stdout, "Tokener agent key is already bound.")
		return err
	}
	return createAndBind(cmd.Context(), deps, target)
}

func regenerateKey(cmd *cobra.Command, deps dependencies) error {
	if _, err := deps.engine.Resolve(cmd.Context(), ""); err != nil {
		return err
	}
	target, err := deps.resolveTarget(cmd)
	if err != nil {
		return err
	}
	noticeCurrentHost(deps.stderr, target.Hostname, target.Ambiguous)
	return createAndBind(cmd.Context(), deps, target)
}

func visibleError(err error) error {
	if err == nil {
		return nil
	}
	var classified *runtime.LatheError
	if errors.As(err, &classified) {
		return err
	}
	var requestError *runtime.HTTPError
	if errors.As(err, &requestError) {
		return err
	}
	return runtime.NewError(
		runtime.CodeGeneral,
		runtime.ExitGeneral,
		err.Error(),
		"check the Tokener agent configuration and retry",
		err,
	)
}

func runAgent(cmd *cobra.Command, deps dependencies, args []string) error {
	harness := ""
	nativeArgs := args
	if len(args) > 0 {
		harness = args[0]
		nativeArgs = args[1:]
		if !knownHarness(harness) {
			return runtime.NewError(
				runtime.CodeUsage,
				runtime.ExitUsage,
				fmt.Sprintf("unknown harness %q; expected one of: %s", harness, strings.Join(harnesses, ", ")),
				"run `tokener agent --help`",
				nil,
			)
		}
	}
	enginePath, err := deps.engine.Resolve(cmd.Context(), harness)
	if err != nil {
		return err
	}
	hostname, _, ambiguous, err := deps.resolveHostname(cmd)
	if err != nil {
		return err
	}
	gateway, err := gatewayEndpointFor(hostname)
	if err != nil {
		return err
	}
	noticeCurrentHost(deps.stderr, hostname, ambiguous)
	key, err := resolveAgentKey(deps.bindings, hostname)
	if err != nil {
		return err
	}
	if key == "" {
		if !deps.interactive() {
			return errors.New("Tokener agent key is not configured; run `tokener agent key login`")
		}
		confirmed, err := confirm(deps.stdin, deps.stderr, "Create and bind a Tokener agent key now?")
		if err != nil {
			return err
		}
		if !confirmed {
			return errors.New("Tokener agent key creation cancelled")
		}
		target, err := deps.resolveTarget(cmd)
		if err != nil {
			return err
		}
		if err := createAndBind(cmd.Context(), deps, target); err != nil {
			return err
		}
		_, err = fmt.Fprintln(deps.stdout, "Run the command again to launch the agent.")
		return err
	}
	stateDir, err := agentStateDir()
	if err != nil {
		return err
	}
	request := hostRequest{
		Harness: harness,
		Gateway: gatewayProfile{
			ProviderID:    "tokener",
			Name:          "Tokener",
			Endpoint:      gateway,
			CredentialEnv: credentialEnv,
		},
		StateDir:         stateDir,
		PermissionPolicy: "standard",
		InstallPolicy:    "prompt",
	}
	return deps.launch(enginePath, request, nativeArgs, key)
}

func createAndBind(ctx context.Context, deps dependencies, target agentTarget) error {
	key, err := deps.createKey(ctx, target)
	if err != nil {
		return err
	}
	if err := deps.bindings.Save(target.Hostname, key); err != nil {
		return err
	}
	_, err = fmt.Fprintln(deps.stdout, "Tokener agent key created and bound.")
	return err
}

func resolveAgentKey(bindings keyBinding, hostname string) (string, error) {
	key, exists, err := bindings.Load(hostname)
	if err != nil || !exists {
		return "", err
	}
	return key, nil
}

func knownHarness(name string) bool {
	for _, harness := range harnesses {
		if name == harness {
			return true
		}
	}
	return false
}

func confirm(input io.Reader, output io.Writer, message string) (bool, error) {
	if _, err := fmt.Fprintf(output, "%s [y/N] ", message); err != nil {
		return false, err
	}
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
