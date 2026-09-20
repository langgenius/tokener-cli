package agent

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/lathe-cli/lathe/pkg/runtime"
	"github.com/spf13/cobra"
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
	key      string
	exists   bool
	err      error
	saved    []string
	hosts    []string
	loadHost string
}

func (binding *fakeBinding) Load(hostname string) (string, bool, error) {
	binding.loadHost = hostname
	return binding.key, binding.exists, binding.err
}

func (binding *fakeBinding) Save(hostname, key string) error {
	binding.hosts = append(binding.hosts, hostname)
	binding.saved = append(binding.saved, key)
	return binding.err
}

type agentFixture struct {
	dependencies
	engine    fakeEngine
	binding   fakeBinding
	output    bytes.Buffer
	errors    bytes.Buffer
	request   hostRequest
	path, key string
	args      []string
	created   int
}

func newAgentFixture(t *testing.T) *agentFixture {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	fixture := &agentFixture{engine: fakeEngine{path: "/engine/rx"}, binding: fakeBinding{key: "bound-key", exists: true}}
	fixture.dependencies = dependencies{
		engine:   &fixture.engine,
		bindings: &fixture.binding,
		resolveHostname: func(*cobra.Command) (string, bool, error) {
			return defaultManagementHostname, false, nil
		},
		resolveTarget: func(*cobra.Command) (agentTarget, error) {
			return agentTarget{Hostname: defaultManagementHostname}, nil
		},
		createKey: func(context.Context, string, runtime.ClientOptions) (string, error) {
			fixture.created++
			return "created-key", nil
		},
		launch: func(path string, request hostRequest, args []string, key string) error {
			fixture.path, fixture.request, fixture.args, fixture.key = path, request, slices.Clone(args), key
			return nil
		},
		interactive: func() bool { return false },
		stdin:       strings.NewReader(""),
		stdout:      &fixture.output,
		stderr:      &fixture.errors,
	}
	return fixture
}

func (fixture *agentFixture) execute(args ...string) error {
	command := newCommand(fixture.dependencies)
	command.SetArgs(append([]string{}, args...))
	command.SilenceErrors = true
	command.SilenceUsage = true
	return command.Execute()
}

func TestAgentLaunch(t *testing.T) {
	t.Setenv(credentialEnv, "ignored-environment-key")
	for _, test := range []struct {
		name, hostname, gateway string
		args                    []string
	}{
		{"native arguments", defaultManagementHostname, defaultGatewayEndpoint, []string{"codex", "resume", "session-1", "--dangerously-bypass-approvals-and-sandbox"}},
		{"selected host", "console-staging.tokener.dev", "https://api-staging.tokener.dev/v1", []string{"codex"}},
		{"interactive picker", defaultManagementHostname, defaultGatewayEndpoint, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentFixture(t)
			fixture.interactive = func() bool { return len(test.args) == 0 }
			fixture.resolveHostname = func(*cobra.Command) (string, bool, error) { return test.hostname, true, nil }
			if err := fixture.execute(test.args...); err != nil {
				t.Fatal(err)
			}
			harness, nativeArgs := "", test.args
			if len(test.args) > 0 {
				harness, nativeArgs = test.args[0], test.args[1:]
			}
			if fixture.created != 0 || !slices.Equal(fixture.engine.calls, []string{harness}) || fixture.path != "/engine/rx" || fixture.key != "bound-key" {
				t.Fatalf("engine/key/create = %#v/%q/%d", fixture.engine, fixture.key, fixture.created)
			}
			if fixture.binding.loadHost != test.hostname || fixture.request.Harness != harness || fixture.request.Gateway.Endpoint != test.gateway || fixture.request.Gateway.ProviderID != "tokener" {
				t.Fatalf("host/request = %s/%#v", fixture.binding.loadHost, fixture.request)
			}
			if fixture.request.InstallPolicy != "prompt" || !slices.Equal(fixture.args, nativeArgs) {
				t.Fatalf("request/arguments = %#v/%v", fixture.request, fixture.args)
			}
		})
	}
}

func TestAgentWithoutHarnessPrintsUsageWhenNoninteractive(t *testing.T) {
	fixture := newAgentFixture(t)
	err := fixture.execute()
	if err == nil || fixture.created != 0 || fixture.path != "" || len(fixture.engine.calls) != 0 {
		t.Fatalf("error/create/launch/engine = %v/%d/%q/%v", err, fixture.created, fixture.path, fixture.engine.calls)
	}
	output := fixture.output.String() + fixture.errors.String() + err.Error()
	for _, harness := range harnesses {
		if !strings.Contains(output, harness) {
			t.Fatalf("missing harness %q in %q", harness, output)
		}
	}
	if strings.Contains(strings.ToLower(output), "rx") {
		t.Fatalf("leaked rx in %q", output)
	}
}

func TestEngineResolutionPrecedesKeyHandling(t *testing.T) {
	t.Setenv(credentialEnv, "ignored-environment-key")
	for _, engineError := range []error{nil, errors.New("incompatible rx")} {
		fixture := newAgentFixture(t)
		fixture.binding = fakeBinding{}
		fixture.engine.err = engineError
		want := "tokener agent key login"
		if engineError != nil {
			fixture.binding.err = errors.New("binding should not be read")
			want = engineError.Error()
		}
		err := fixture.execute("claude")
		if err == nil || !strings.Contains(err.Error(), want) || !slices.Equal(fixture.engine.calls, []string{"claude"}) || fixture.created != 0 || fixture.path != "" {
			t.Fatalf("error/engine/create/launch = %v/%v/%d/%q", err, fixture.engine.calls, fixture.created, fixture.path)
		}
	}
}

func TestInteractiveMissingKeyCreatesBindingAndExits(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding = fakeBinding{}
	fixture.interactive = func() bool { return true }
	fixture.stdin = strings.NewReader("yes\n")
	if err := fixture.execute("pi"); err != nil {
		t.Fatal(err)
	}
	if fixture.created != 1 || !slices.Equal(fixture.binding.saved, []string{"created-key"}) || !slices.Equal(fixture.binding.hosts, []string{defaultManagementHostname}) || fixture.path != "" {
		t.Fatalf("created/binding/launch = %d/%#v/%q", fixture.created, fixture.binding, fixture.path)
	}
	if output := fixture.output.String(); !strings.Contains(output, "created and bound") || !strings.Contains(output, "Run the command again") {
		t.Fatalf("output = %q", output)
	}
}

func TestKeyStatusDoesNotMutateOrRevealKey(t *testing.T) {
	for _, key := range []string{"", "sk-abcdefghijklmnopqrstuvwxyz"} {
		fixture := newAgentFixture(t)
		fixture.binding = fakeBinding{key: key, exists: key != ""}
		if err := fixture.execute("key", "status"); err != nil {
			t.Fatal(err)
		}
		if fixture.created != 0 || fixture.path != "" || len(fixture.engine.calls) != 0 {
			t.Fatalf("status create/launch/engine = %d/%q/%v", fixture.created, fixture.path, fixture.engine.calls)
		}
		prefix := "bound: false\n"
		if key != "" {
			prefix = "bound: true\nprefix: sk-abcde\n"
		}
		if output := fixture.output.String(); output != prefix+"host: "+defaultManagementHostname+"\ngateway: "+defaultGatewayEndpoint+"\n" {
			t.Fatalf("status output = %q", output)
		}
	}
}

func TestKeyCommandsPreserveExistingBindingUnlessRegenerated(t *testing.T) {
	fixture := newAgentFixture(t)
	keyCommand, _, err := newCommand(fixture.dependencies).Find([]string{"key"})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, child := range keyCommand.Commands() {
		names = append(names, child.Name())
	}
	if !slices.Equal(names, []string{"login", "regenerate", "status"}) {
		t.Fatalf("key commands = %v", names)
	}
	for _, test := range []struct {
		command string
		bound   bool
		created int
	}{
		{"login", true, 0},
		{"login", false, 1},
		{"regenerate", true, 1},
		{"regenerate", false, 1},
	} {
		fixture := newAgentFixture(t)
		fixture.binding.exists = test.bound
		if err := fixture.execute("key", test.command); err != nil {
			t.Fatal(err)
		}
		if fixture.created != test.created || !slices.Equal(fixture.engine.calls, []string{""}) {
			t.Fatalf("%#v: created/engine = %d/%v", test, fixture.created, fixture.engine.calls)
		}
		if test.created != 0 && (!slices.Equal(fixture.binding.saved, []string{"created-key"}) || !slices.Equal(fixture.binding.hosts, []string{defaultManagementHostname})) {
			t.Fatalf("%#v: binding = %#v", test, fixture.binding)
		}
	}
}
