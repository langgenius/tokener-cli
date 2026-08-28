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

const (
	gatewayEndpoint = "https://api.tokener.dev/v1"
	credentialEnv   = "TOKENER_API_KEY"
)

var harnesses = []string{"claude", "codex", "opencode", "pi", "dsh", "kimi"}

type engineResolver interface {
	Resolve(context.Context, string) (string, error)
}

type keyBinding interface {
	Load() (string, bool, error)
	Save(string) error
}

type dependencies struct {
	engine      engineResolver
	bindings    keyBinding
	createKey   func(context.Context) (string, error)
	launch      func(string, hostRequest, []string, string) error
	interactive func() bool
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
}

func NewCommand() *cobra.Command {
	bindings := newFileBinding()
	return newCommand(dependencies{
		engine:    newEmbeddedEngine(),
		bindings:  bindings,
		createKey: createAgentKey,
		launch:    launchEngine,
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
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			return visibleError(runAgent(cmd.Context(), deps, args))
		},
	}
	cmd.AddCommand(newKeyCommand(deps))
	return cmd
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
			return visibleError(loginKey(cmd.Context(), deps))
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "regenerate",
		Short: "Create and bind a new agent key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return visibleError(regenerateKey(cmd.Context(), deps))
		},
	})
	return cmd
}

func loginKey(ctx context.Context, deps dependencies) error {
	if _, err := deps.engine.Resolve(ctx, ""); err != nil {
		return err
	}
	if _, exists, err := deps.bindings.Load(); err != nil {
		return err
	} else if exists {
		_, err = fmt.Fprintln(deps.stdout, "Tokener agent key is already bound.")
		return err
	}
	return createAndBind(ctx, deps)
}

func regenerateKey(ctx context.Context, deps dependencies) error {
	if _, err := deps.engine.Resolve(ctx, ""); err != nil {
		return err
	}
	return createAndBind(ctx, deps)
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

func runAgent(ctx context.Context, deps dependencies, args []string) error {
	harness := ""
	nativeArgs := args
	if len(args) > 0 {
		harness = args[0]
		nativeArgs = args[1:]
		if !knownHarness(harness) {
			return fmt.Errorf("unknown harness %q; expected one of: %s", harness, strings.Join(harnesses, ", "))
		}
	}
	enginePath, err := deps.engine.Resolve(ctx, harness)
	if err != nil {
		return err
	}
	key, err := resolveAgentKey(deps.bindings)
	if err != nil {
		return err
	}
	if key == "" {
		if !deps.interactive() {
			return errors.New("Tokener agent key is not configured; set TOKENER_API_KEY or run `tokener agent key login`")
		}
		confirmed, err := confirm(deps.stdin, deps.stderr, "Create and bind a Tokener agent key now?")
		if err != nil {
			return err
		}
		if !confirmed {
			return errors.New("Tokener agent key creation cancelled")
		}
		if err := createAndBind(ctx, deps); err != nil {
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
			Endpoint:      gatewayEndpoint,
			CredentialEnv: credentialEnv,
		},
		StateDir:         stateDir,
		PermissionPolicy: "standard",
		InstallPolicy:    "prompt",
	}
	return deps.launch(enginePath, request, nativeArgs, key)
}

func createAndBind(ctx context.Context, deps dependencies) error {
	key, err := deps.createKey(ctx)
	if err != nil {
		return err
	}
	if err := deps.bindings.Save(key); err != nil {
		return err
	}
	_, err = fmt.Fprintln(deps.stdout, "Tokener agent key created and bound.")
	return err
}

func resolveAgentKey(bindings keyBinding) (string, error) {
	if key := os.Getenv(credentialEnv); key != "" {
		return key, nil
	}
	key, exists, err := bindings.Load()
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
