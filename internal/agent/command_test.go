package agent

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

type fakeEngine struct {
	path  string
	err   error
	calls []string
}

func (engine *fakeEngine) Resolve(_ context.Context, harness string) (string, error) {
	engine.calls = append(engine.calls, harness)
	return engine.path, engine.err
}

type fakeBinding struct {
	key    string
	exists bool
	err    error
	saved  []string
}

func (binding *fakeBinding) Load() (string, bool, error) {
	return binding.key, binding.exists, binding.err
}

func (binding *fakeBinding) Save(key string) error {
	binding.saved = append(binding.saved, key)
	return binding.err
}

type launchCall struct {
	path    string
	request hostRequest
	args    []string
	key     string
}

func testDependencies(engine *fakeEngine, binding *fakeBinding) (dependencies, *launchCall, *int) {
	call := &launchCall{}
	created := 0
	return dependencies{
		engine:   engine,
		bindings: binding,
		createKey: func(context.Context) (string, error) {
			created++
			return "created-key", nil
		},
		launch: func(path string, request hostRequest, args []string, key string) error {
			call.path = path
			call.request = request
			call.args = slices.Clone(args)
			call.key = key
			return nil
		},
		interactive: func() bool { return false },
		stdin:       strings.NewReader(""),
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
	}, call, &created
}

func executeAgent(t *testing.T, deps dependencies, args ...string) error {
	t.Helper()
	command := newCommand(deps)
	if args == nil {
		args = make([]string, 0)
	}
	command.SetArgs(args)
	command.SilenceErrors = true
	command.SilenceUsage = true
	return command.Execute()
}

func TestAgentLaunchUsesFixedGatewayAndNativeArguments(t *testing.T) {
	t.Setenv(credentialEnv, "environment-key")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	engine := &fakeEngine{path: "/engine/rx"}
	binding := &fakeBinding{key: "bound-key", exists: true}
	deps, call, _ := testDependencies(engine, binding)

	if err := executeAgent(t, deps, "codex", "resume", "session-1", "--dangerously-bypass-approvals-and-sandbox"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(engine.calls, []string{"codex"}) {
		t.Fatalf("engine calls = %v", engine.calls)
	}
	if call.path != "/engine/rx" || call.key != "environment-key" {
		t.Fatalf("launch path/key = %q/%q", call.path, call.key)
	}
	if call.request.Harness != "codex" || call.request.Gateway.Endpoint != gatewayEndpoint || call.request.Gateway.ProviderID != "tokener" {
		t.Fatalf("request = %#v", call.request)
	}
	if call.request.PermissionPolicy != "standard" || call.request.InstallPolicy != "prompt" {
		t.Fatalf("policies = %q/%q", call.request.PermissionPolicy, call.request.InstallPolicy)
	}
	if !slices.Equal(call.args, []string{"resume", "session-1", "--dangerously-bypass-approvals-and-sandbox"}) {
		t.Fatalf("native args = %v", call.args)
	}
}

func TestAgentWithoutHarnessLeavesHostedRequestHarnessEmpty(t *testing.T) {
	t.Setenv(credentialEnv, "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	engine := &fakeEngine{path: "/engine/rx"}
	binding := &fakeBinding{key: "bound-key", exists: true}
	deps, call, _ := testDependencies(engine, binding)

	if err := executeAgent(t, deps); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(engine.calls, []string{""}) || call.request.Harness != "" {
		t.Fatalf("engine calls/request harness = %v/%q", engine.calls, call.request.Harness)
	}
}

func TestMissingKeyResolvesEngineBeforeNoninteractiveFailure(t *testing.T) {
	t.Setenv(credentialEnv, "")
	engine := &fakeEngine{path: "/engine/rx"}
	binding := &fakeBinding{}
	deps, call, created := testDependencies(engine, binding)

	err := executeAgent(t, deps, "claude")
	if err == nil || !strings.Contains(err.Error(), "tokener agent key login") {
		t.Fatalf("error = %v", err)
	}
	if !slices.Equal(engine.calls, []string{"claude"}) || *created != 0 || call.path != "" {
		t.Fatalf("engine/created/launch = %v/%d/%q", engine.calls, *created, call.path)
	}
}

func TestInteractiveMissingKeyCreatesBindingAndExits(t *testing.T) {
	t.Setenv(credentialEnv, "")
	engine := &fakeEngine{path: "/engine/rx"}
	binding := &fakeBinding{}
	deps, call, created := testDependencies(engine, binding)
	output := &bytes.Buffer{}
	deps.interactive = func() bool { return true }
	deps.stdin = strings.NewReader("yes\n")
	deps.stdout = output
	deps.stderr = &bytes.Buffer{}

	if err := executeAgent(t, deps, "pi"); err != nil {
		t.Fatal(err)
	}
	if *created != 1 || !slices.Equal(binding.saved, []string{"created-key"}) || call.path != "" {
		t.Fatalf("created/saved/launch = %d/%v/%q", *created, binding.saved, call.path)
	}
	if !strings.Contains(output.String(), "created and bound") || !strings.Contains(output.String(), "Run the command again") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestOverrideFailurePrecedesKeyHandling(t *testing.T) {
	t.Setenv(credentialEnv, "")
	engine := &fakeEngine{err: errors.New("incompatible rx")}
	binding := &fakeBinding{err: errors.New("binding should not be read")}
	deps, _, created := testDependencies(engine, binding)

	err := executeAgent(t, deps, "opencode")
	if err == nil || err.Error() != "incompatible rx" || *created != 0 {
		t.Fatalf("error/created = %v/%d", err, *created)
	}
}

func TestKeySubcommandsAreOnlyLoginAndRegenerate(t *testing.T) {
	engine := &fakeEngine{path: "/engine/rx"}
	binding := &fakeBinding{key: "bound-key", exists: true}
	deps, _, created := testDependencies(engine, binding)
	command := newCommand(deps)
	keyCommand, _, err := command.Find([]string{"key"})
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(keyCommand.Commands()))
	for _, child := range keyCommand.Commands() {
		names = append(names, child.Name())
	}
	if !slices.Equal(names, []string{"login", "regenerate"}) {
		t.Fatalf("key commands = %v", names)
	}

	if err := executeAgent(t, deps, "key", "login"); err != nil {
		t.Fatal(err)
	}
	if *created != 0 || !slices.Equal(engine.calls, []string{""}) {
		t.Fatalf("login created/engine = %d/%v", *created, engine.calls)
	}
	binding.exists = false
	if err := executeAgent(t, deps, "key", "regenerate"); err != nil {
		t.Fatal(err)
	}
	if *created != 1 || !slices.Equal(binding.saved, []string{"created-key"}) {
		t.Fatalf("regenerate created/saved = %d/%v", *created, binding.saved)
	}
}
