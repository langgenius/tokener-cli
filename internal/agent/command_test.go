package agent

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

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
	binding  agentBinding
	exists   bool
	err      error
	saved    []agentBinding
	hosts    []string
	loadHost string
}

func (binding *fakeBinding) Load(hostname string) (agentBinding, bool, error) {
	binding.loadHost = hostname
	return binding.binding, binding.exists, binding.err
}

func (binding *fakeBinding) Save(hostname string, stored agentBinding) error {
	binding.hosts = append(binding.hosts, hostname)
	binding.saved = append(binding.saved, stored)
	return binding.err
}

func (binding *fakeBinding) savedKeys() []string {
	keys := make([]string, 0, len(binding.saved))
	for _, stored := range binding.saved {
		keys = append(keys, stored.Key)
	}
	return keys
}

// testIdentity is the machine identity injected into fixtures so key names are
// deterministic without touching the platform identifier.
var testIdentity = machineIdentity{Hostname: "test-box", Digest: "abc123"}

type agentFixture struct {
	dependencies
	engine    fakeEngine
	binding   fakeBinding
	keyAPI    fakeKeys
	output    bytes.Buffer
	errors    bytes.Buffer
	request   hostRequest
	path, key string
	args      []string
}

func newAgentFixture(t *testing.T) *agentFixture {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	fixture := &agentFixture{
		engine:  fakeEngine{path: "/engine/rx"},
		binding: fakeBinding{binding: agentBinding{Key: "bound-key"}, exists: true},
	}
	fixture.keyAPI = fakeKeys{secret: "created-key"}
	fixture.dependencies = dependencies{
		engine:   &fixture.engine,
		bindings: &fixture.binding,
		keys:     &fixture.keyAPI,
		identity: func() (machineIdentity, error) { return testIdentity, nil },
		resolveHostname: func(*cobra.Command) (string, bool, error) {
			return defaultManagementHostname, false, nil
		},
		resolveTarget: func(*cobra.Command) (agentTarget, error) {
			return agentTarget{Hostname: defaultManagementHostname}, nil
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

func (fixture *agentFixture) created() int {
	return len(fixture.keyAPI.created)
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
			if fixture.created() != 0 || !slices.Equal(fixture.engine.calls, []string{harness}) || fixture.path != "/engine/rx" || fixture.key != "bound-key" {
				t.Fatalf("engine/key/create = %#v/%q/%d", fixture.engine, fixture.key, fixture.created())
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
	if err == nil || fixture.created() != 0 || fixture.path != "" || len(fixture.engine.calls) != 0 {
		t.Fatalf("error/create/launch/engine = %v/%d/%q/%v", err, fixture.created(), fixture.path, fixture.engine.calls)
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
		if err == nil || !strings.Contains(err.Error(), want) || !slices.Equal(fixture.engine.calls, []string{"claude"}) || fixture.created() != 0 || fixture.path != "" {
			t.Fatalf("error/engine/create/launch = %v/%v/%d/%q", err, fixture.engine.calls, fixture.created(), fixture.path)
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
	if fixture.created() != 1 || !slices.Equal(fixture.binding.savedKeys(), []string{"created-key-1"}) || !slices.Equal(fixture.binding.hosts, []string{defaultManagementHostname}) || fixture.path != "" {
		t.Fatalf("created/binding/launch = %d/%#v/%q", fixture.created(), fixture.binding, fixture.path)
	}
	if output := fixture.output.String(); !strings.Contains(output, "created and bound") || !strings.Contains(output, "Run the command again") {
		t.Fatalf("output = %q", output)
	}
}

func TestKeyStatusDoesNotMutateOrRevealKey(t *testing.T) {
	for _, test := range []struct {
		name    string
		binding agentBinding
		want    string
	}{
		{"unbound", agentBinding{}, "bound: false\n"},
		{
			"legacy binding without name or id",
			agentBinding{Key: "sk-abcdefghijklmnopqrstuvwxyz"},
			"bound: true\nprefix: sk-abcde\n",
		},
		{
			"per-machine binding",
			agentBinding{Key: "sk-abcdefghijklmnopqrstuvwxyz", KeyID: "key-7", Name: "Tokener Agent CLI · box-abc123"},
			"bound: true\nprefix: sk-abcde\nname: Tokener Agent CLI · box-abc123\nkey-id: key-7\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentFixture(t)
			fixture.binding = fakeBinding{binding: test.binding, exists: test.binding.Key != ""}
			if err := fixture.execute("key", "status"); err != nil {
				t.Fatal(err)
			}
			if fixture.created() != 0 || fixture.path != "" || len(fixture.engine.calls) != 0 {
				t.Fatalf("status create/launch/engine = %d/%q/%v", fixture.created(), fixture.path, fixture.engine.calls)
			}
			if len(fixture.keyAPI.revealed) != 0 || len(fixture.keyAPI.revoked) != 0 || fixture.keyAPI.listed != 0 {
				t.Fatalf("status touched the API: %#v", fixture.keyAPI)
			}
			want := test.want + "host: " + defaultManagementHostname + "\ngateway: " + defaultGatewayEndpoint + "\n"
			if output := fixture.output.String(); output != want {
				t.Fatalf("status output = %q, want %q", output, want)
			}
		})
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
		if fixture.created() != test.created || !slices.Equal(fixture.engine.calls, []string{""}) {
			t.Fatalf("%#v: created/engine = %d/%v", test, fixture.created(), fixture.engine.calls)
		}
		if test.created != 0 && (!slices.Equal(fixture.binding.savedKeys(), []string{"created-key-1"}) || !slices.Equal(fixture.binding.hosts, []string{defaultManagementHostname})) {
			t.Fatalf("%#v: binding = %#v", test, fixture.binding)
		}
	}
}
