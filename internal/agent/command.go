package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
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
	Load(hostname string) (bindingDocument, bool, error)
	Save(hostname string, document bindingDocument) error
}

type dependencies struct {
	engine          engineResolver
	bindings        keyBinding
	resolveHostname func(*cobra.Command) (string, bool, error)
	resolveTarget   func(*cobra.Command) (agentTarget, error)
	createKey       func(context.Context, string, runtime.ClientOptions) (createdKey, error)
	revokeKey       func(context.Context, string, string, runtime.ClientOptions) error
	launch          func(string, hostRequest, []string, string) error
	interactive     func() bool
	stdin           io.Reader
	stdout          io.Writer
	stderr          io.Writer
}

func NewCommand() *cobra.Command {
	return newCommand(dependencies{
		engine:          newEmbeddedEngine(),
		bindings:        fileBinding{},
		resolveHostname: resolveManagementHostname,
		resolveTarget:   resolveAgentTarget,
		createKey:       createKeyRequest,
		revokeKey:       revokeKeyRequest,
		launch:          launchEngine,
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
		Long:               "Run coding agents through the Tokener Gateway.\n\nAvailable harnesses: " + strings.Join(harnesses, ", "),
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
		if !slices.Contains(harnesses, harness) {
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
	hostname, ambiguous, err := deps.resolveHostname(cmd)
	if err != nil {
		return err
	}
	gateway, err := gatewayEndpointFor(hostname)
	if err != nil {
		return err
	}
	noticeCurrentHost(deps.stderr, hostname, ambiguous)
	document, exists, err := deps.bindings.Load(hostname)
	if err != nil {
		return err
	}
	if !exists || document.Key == "" {
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
		StateDir:      stateDir,
		InstallPolicy: "prompt",
	}
	return deps.launch(enginePath, request, nativeArgs, document.Key)
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
