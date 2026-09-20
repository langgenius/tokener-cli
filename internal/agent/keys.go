package agent

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"github.com/lathe-cli/lathe/pkg/runtime"
	"github.com/spf13/cobra"
)

// maxKeyNameAttempts bounds how many names a single bind tries before giving
// up: the canonical name, then two numbered variants.
const maxKeyNameAttempts = 3

func newKeyCommand(deps dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage the local Tokener agent key binding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	for _, action := range []struct {
		name, description string
		run               func(*cobra.Command) error
	}{
		{"login", "Bind this machine's agent key, reusing it when it already exists", func(cmd *cobra.Command) error {
			return loginKey(cmd, deps)
		}},
		{"regenerate", "Revoke this machine's agent key and bind a fresh one", func(cmd *cobra.Command) error {
			return regenerateKey(cmd, deps)
		}},
		{"status", "Show the local Tokener agent key binding", func(cmd *cobra.Command) error {
			return statusKey(cmd, deps)
		}},
	} {
		cmd.AddCommand(&cobra.Command{
			Use:   action.name,
			Short: action.description,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return visibleError(action.run(cmd))
			},
		})
	}
	return cmd
}

func statusKey(cmd *cobra.Command, deps dependencies) error {
	hostname, _, err := deps.resolveHostname(cmd)
	if err != nil {
		return err
	}
	gateway, err := gatewayEndpointFor(hostname)
	if err != nil {
		return err
	}
	stored, exists, err := deps.bindings.Load(hostname)
	if err != nil {
		return err
	}
	if !exists {
		_, err = fmt.Fprintf(deps.stdout, "bound: false\nhost: %s\ngateway: %s\n", hostname, gateway)
		return err
	}
	if _, err := fmt.Fprintf(deps.stdout, "bound: true\nprefix: %s\n", stored.Key[:min(8, len(stored.Key))]); err != nil {
		return err
	}
	// Bindings written before keys were named per machine carry neither field.
	for _, field := range []struct{ label, value string }{{"name", stored.Name}, {"key-id", stored.KeyID}} {
		if field.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(deps.stdout, "%s: %s\n", field.label, field.value); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(deps.stdout, "host: %s\ngateway: %s\n", hostname, gateway)
	return err
}

// loginKey binds an agent key for this machine. It is idempotent: a key already
// carrying this machine's name is adopted rather than duplicated, which is what
// lets a second machine, or a retry after an interrupted login, succeed.
func loginKey(cmd *cobra.Command, deps dependencies) error {
	target, err := prepareKeyCommand(cmd, deps)
	if err != nil {
		return err
	}
	if _, exists, err := deps.bindings.Load(target.Hostname); err != nil {
		return err
	} else if exists {
		_, err = fmt.Fprintln(deps.stdout, "Tokener agent key is already bound.")
		return err
	}
	return claimOrCreateKey(cmd.Context(), deps, target)
}

// regenerateKey rotates this machine's key: the current one is revoked, which
// frees the name, and a new key is created under the same name.
func regenerateKey(cmd *cobra.Command, deps dependencies) error {
	target, err := prepareKeyCommand(cmd, deps)
	if err != nil {
		return err
	}
	identity, err := deps.identity()
	if err != nil {
		return err
	}
	revoked, err := revokeCurrentKey(cmd.Context(), deps, target, identity)
	if err != nil {
		return err
	}
	err = createKey(cmd.Context(), deps, target, identity, false)
	if err != nil && revoked {
		// The old key is already gone, so the stored binding is now dead and
		// the machine cannot run agents until a new key is bound.
		return rotationInterruptedError(err)
	}
	return err
}

func prepareKeyCommand(cmd *cobra.Command, deps dependencies) (agentTarget, error) {
	if _, err := deps.engine.Resolve(cmd.Context(), ""); err != nil {
		return agentTarget{}, err
	}
	target, err := deps.resolveTarget(cmd)
	if err != nil {
		return agentTarget{}, err
	}
	noticeCurrentHost(deps.stderr, target.Hostname, target.Ambiguous)
	return target, nil
}

// claimOrCreateKey adopts this machine's existing key when there is one, and
// otherwise creates it.
func claimOrCreateKey(ctx context.Context, deps dependencies, target agentTarget) error {
	identity, err := deps.identity()
	if err != nil {
		return err
	}
	claimed, err := claimKey(ctx, deps, target, identity, identity.keyName(1))
	if err != nil {
		return err
	}
	if claimed {
		return nil
	}
	return createKey(ctx, deps, target, identity, true)
}

// claimKey binds the key named name when it already exists. The name embeds
// this machine's digest, so such a key can only have been created by this
// machine on an earlier run; adopting it cannot take another machine's key.
// A key whose secret cannot be revealed is revoked to free the name, and the
// caller is told to create a replacement by a false result.
func claimKey(ctx context.Context, deps dependencies, target agentTarget, identity machineIdentity, name string) (bool, error) {
	records, err := deps.keys.List(ctx, target.Hostname, target.Options)
	if err != nil {
		return false, err
	}
	record, found := findKeyByName(records, name)
	if !found {
		return false, nil
	}
	key, err := deps.keys.Reveal(ctx, target.Hostname, record.ID, target.Options)
	if err != nil {
		_, _ = fmt.Fprintf(
			deps.stderr,
			"Existing Tokener agent key %q could not be read back; revoking and replacing it.\n",
			name,
		)
		if err := deps.keys.Revoke(ctx, target.Hostname, record.ID, target.Options); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := bindKey(deps, target, identity, record, name, key); err != nil {
		return false, err
	}
	_, err = fmt.Fprintln(deps.stdout, "Tokener agent key for this machine reused and bound.")
	return true, err
}

// createKey creates and binds a key for this machine. When claim is set, every
// name collision is first retried as a claim, so a login that raced with another
// run on this machine, or that resumes one interrupted after it had already
// fallen back to a numbered name, adopts that key instead of stepping over it
// and leaving it orphaned.
func createKey(ctx context.Context, deps dependencies, target agentTarget, identity machineIdentity, claim bool) error {
	for attempt := 1; attempt <= maxKeyNameAttempts; attempt++ {
		name := identity.keyName(attempt)
		record, key, err := deps.keys.Create(ctx, target.Hostname, name, target.Options)
		if err == nil {
			if err := bindKey(deps, target, identity, record, name, key); err != nil {
				return err
			}
			_, err = fmt.Fprintln(deps.stdout, "Tokener agent key created and bound.")
			return err
		}
		if !isDuplicateKeyName(err) {
			return err
		}
		if claim {
			claimed, claimErr := claimKey(ctx, deps, target, identity, name)
			if claimErr != nil {
				return claimErr
			}
			if claimed {
				return nil
			}
		}
	}
	return duplicateKeyNameError(identity.keyName(1))
}

// revokeCurrentKey revokes the key this machine is using, if it can be found,
// and reports whether a key was actually taken out of service. The binding id
// is authoritative. Without one the key is located by exact name: the recorded
// name, then this machine's current name, then the single name every key
// carried before names became per-machine.
func revokeCurrentKey(ctx context.Context, deps dependencies, target agentTarget, identity machineIdentity) (bool, error) {
	stored, exists, err := deps.bindings.Load(target.Hostname)
	if err != nil {
		return false, err
	}
	id := ""
	if exists {
		id = stored.KeyID
	}
	if id == "" {
		records, err := deps.keys.List(ctx, target.Hostname, target.Options)
		if err != nil {
			return false, err
		}
		record, found := findKeyByNames(records, stored.Name, identity.keyName(1), agentKeyNamePrefix)
		if !found {
			return false, nil
		}
		id = record.ID
	}
	switch err := deps.keys.Revoke(ctx, target.Hostname, id, target.Options); {
	case err == nil:
		return true, nil
	case isKeyAlreadyGone(err):
		// Someone already revoked it, for instance with `tokener keys revoke`.
		// The name is free, so rotation can continue.
		return false, nil
	default:
		return false, err
	}
}

func bindKey(deps dependencies, target agentTarget, identity machineIdentity, record keyRecord, name, key string) error {
	return deps.bindings.Save(target.Hostname, agentBinding{
		Key:       key,
		KeyID:     record.ID,
		Name:      cmp.Or(record.Name, name),
		MachineID: identity.Digest,
	})
}

// rotationInterruptedError reports a rotation that revoked the old key but
// could not create its replacement, so the local binding no longer works.
func rotationInterruptedError(cause error) error {
	message := cause.Error()
	var classified *runtime.LatheError
	if errors.As(cause, &classified) {
		message = classified.Message
	}
	return runtime.NewError(
		runtime.CodeGeneral,
		runtime.ExitGeneral,
		"this machine's previous Tokener agent key was revoked but its replacement could not be created: "+message,
		// The stale binding makes `key login` report that a key is already
		// bound, so rotation has to be retried with `key regenerate`.
		"run `tokener agent key regenerate` again to finish binding a new key",
		cause,
	)
}

// duplicateKeyNameError explains a collision the CLI could not resolve on its
// own. It happens when another key in the organization already holds the name,
// for instance one owned by a different user and therefore invisible to a
// member's key listing.
func duplicateKeyNameError(name string) error {
	return runtime.NewError(
		runtime.CodeGeneral,
		runtime.ExitGeneral,
		fmt.Sprintf(
			"a Tokener API key named %q already exists in this organization, and the numbered alternatives are taken too, so this machine's agent key could not be created",
			name,
		),
		"run `tokener agent key regenerate` to replace this machine's key, "+
			"or find the conflicting key with `tokener keys list`, run `tokener keys revoke <key-id>`, then `tokener agent key login`",
		nil,
	)
}
