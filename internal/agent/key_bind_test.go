package agent

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/lathe-cli/lathe/pkg/runtime"
)

// fakeKeys models the console key API, including the server's rule that a name
// is unique per organization among keys that are not revoked.
type fakeKeys struct {
	records []keyRecord
	// hidden names are taken in the organization but absent from List, which is
	// what a member sees for keys owned by another user.
	hidden    []string
	secrets   map[string]string
	secret    string
	revealErr error
	listErr   error
	revokeErr error

	listed   int
	created  []string
	revealed []string
	revoked  []string
}

func (keys *fakeKeys) List(_ context.Context, _ string, _ runtime.ClientOptions) ([]keyRecord, error) {
	keys.listed++
	if keys.listErr != nil {
		return nil, keys.listErr
	}
	return slices.Clone(keys.records), nil
}

func (keys *fakeKeys) Create(_ context.Context, _ string, name string, _ runtime.ClientOptions) (keyRecord, string, error) {
	keys.created = append(keys.created, name)
	_, visible := findKeyByName(keys.records, name)
	if visible || slices.Contains(keys.hidden, name) {
		return keyRecord{}, "", duplicateKeyNameResponse()
	}
	secret := fmt.Sprintf("%s-%d", keys.secret, len(keys.created))
	record := keyRecord{
		ID:     fmt.Sprintf("key-%d", len(keys.created)),
		Name:   name,
		Status: "active",
	}
	keys.records = append(keys.records, record)
	if keys.secrets == nil {
		keys.secrets = map[string]string{}
	}
	keys.secrets[record.ID] = secret
	return record, secret, nil
}

func (keys *fakeKeys) Reveal(_ context.Context, _ string, id string, _ runtime.ClientOptions) (string, error) {
	keys.revealed = append(keys.revealed, id)
	if keys.revealErr != nil {
		return "", keys.revealErr
	}
	secret, exists := keys.secrets[id]
	if !exists {
		return "", fmt.Errorf("no secret for %q", id)
	}
	return secret, nil
}

func (keys *fakeKeys) Revoke(_ context.Context, _ string, id string, _ runtime.ClientOptions) error {
	keys.revoked = append(keys.revoked, id)
	if keys.revokeErr != nil {
		return keys.revokeErr
	}
	for index := range keys.records {
		if keys.records[index].ID == id {
			keys.records[index].Status = "revoked"
		}
	}
	return nil
}

// seed registers a key that already exists on the server, as one created by an
// earlier run would be.
func (keys *fakeKeys) seed(id, name, secret string) keyRecord {
	record := keyRecord{ID: id, Name: name, Status: "active"}
	keys.records = append(keys.records, record)
	if keys.secrets == nil {
		keys.secrets = map[string]string{}
	}
	keys.secrets[id] = secret
	return record
}

func duplicateKeyNameResponse() error {
	return &runtime.HTTPError{
		Method:      http.MethodPost,
		URL:         "https://console.tokener.dev" + keysPath,
		Status:      http.StatusBadRequest,
		ContentType: "application/json",
		Body:        []byte(`{"error":"duplicate_key_name"}`),
	}
}

func canonicalTestKeyName() string {
	return testIdentity.keyName(1)
}

func TestLoginWithoutExistingKeyCreatesExactlyOne(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.exists = false
	if err := fixture.execute("key", "login"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.created, []string{canonicalTestKeyName()}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
	if len(fixture.keyAPI.revoked) != 0 || len(fixture.keyAPI.revealed) != 0 {
		t.Fatalf("revoked/revealed = %q/%q", fixture.keyAPI.revoked, fixture.keyAPI.revealed)
	}
	stored := fixture.binding.saved
	if len(stored) != 1 || stored[0].KeyID != "key-1" || stored[0].Name != canonicalTestKeyName() {
		t.Fatalf("binding = %#v", stored)
	}
	if stored[0].MachineID != testIdentity.Digest || stored[0].Key != "created-key-1" {
		t.Fatalf("binding = %#v", stored)
	}
}

func TestLoginClaimsExistingKeyForThisMachineWithoutCreating(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.exists = false
	fixture.keyAPI.seed("key-existing", canonicalTestKeyName(), "sk-existing-secret")

	if err := fixture.execute("key", "login"); err != nil {
		t.Fatal(err)
	}
	if len(fixture.keyAPI.created) != 0 || len(fixture.keyAPI.revoked) != 0 {
		t.Fatalf("created/revoked = %q/%q", fixture.keyAPI.created, fixture.keyAPI.revoked)
	}
	if !slices.Equal(fixture.keyAPI.revealed, []string{"key-existing"}) {
		t.Fatalf("revealed = %q", fixture.keyAPI.revealed)
	}
	want := agentBinding{
		Key:       "sk-existing-secret",
		KeyID:     "key-existing",
		Name:      canonicalTestKeyName(),
		MachineID: testIdentity.Digest,
	}
	if !slices.Equal(fixture.binding.saved, []agentBinding{want}) {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
	if output := fixture.output.String(); !strings.Contains(output, "reused and bound") {
		t.Fatalf("output = %q", output)
	}
}

func TestLoginRevokesAndRecreatesWhenRevealFails(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.exists = false
	fixture.keyAPI.seed("key-stale", canonicalTestKeyName(), "sk-unreadable")
	fixture.keyAPI.revealErr = fmt.Errorf("api_key_secret_unavailable")

	if err := fixture.execute("key", "login"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.revoked, []string{"key-stale"}) {
		t.Fatalf("revoked = %q", fixture.keyAPI.revoked)
	}
	// The revoke frees the name, so the replacement keeps the canonical one.
	if !slices.Equal(fixture.keyAPI.created, []string{canonicalTestKeyName()}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
	if len(fixture.binding.saved) != 1 || fixture.binding.saved[0].Name != canonicalTestKeyName() {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
}

func TestLoginRelooksUpOnceAndClaimsAfterDuplicate(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.exists = false
	// The key appears only after the first lookup, as a concurrent run on this
	// machine would make it.
	appear := fixture.keyAPI.seed("key-raced", canonicalTestKeyName(), "sk-raced")
	fixture.keyAPI.records = nil
	fixture.keyAPI.hidden = []string{canonicalTestKeyName()}
	restore := func() {
		fixture.keyAPI.records = []keyRecord{appear}
		fixture.keyAPI.hidden = nil
	}
	fixture.dependencies.identity = func() (machineIdentity, error) { return testIdentity, nil }
	fixture.keyAPI.listErr = nil

	// Make the key visible once the create has failed with a duplicate.
	original := fixture.dependencies.keys
	fixture.dependencies.keys = &revealAfterCreate{keyClient: original, onCreate: restore}

	if err := fixture.execute("key", "login"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.created, []string{canonicalTestKeyName()}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
	if !slices.Equal(fixture.keyAPI.revealed, []string{"key-raced"}) {
		t.Fatalf("revealed = %q", fixture.keyAPI.revealed)
	}
	if fixture.keyAPI.listed != 2 {
		t.Fatalf("lookups = %d", fixture.keyAPI.listed)
	}
	if len(fixture.binding.saved) != 1 || fixture.binding.saved[0].Key != "sk-raced" {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
}

// revealAfterCreate lets a test change server state at the moment a create is
// rejected, modelling a key that another process committed in between.
type revealAfterCreate struct {
	keyClient
	onCreate func()
	fired    bool
}

func (client *revealAfterCreate) Create(ctx context.Context, hostname, name string, options runtime.ClientOptions) (keyRecord, string, error) {
	record, secret, err := client.keyClient.Create(ctx, hostname, name, options)
	if err != nil && !client.fired {
		client.fired = true
		client.onCreate()
	}
	return record, secret, err
}

func TestLoginSuffixesThenFailsWithRecoveryInstructions(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.exists = false
	// Every candidate name is taken by keys this caller cannot see.
	fixture.keyAPI.hidden = []string{
		testIdentity.keyName(1),
		testIdentity.keyName(2),
		testIdentity.keyName(3),
	}

	err := fixture.execute("key", "login")
	if err == nil {
		t.Fatal("collision on every name was accepted")
	}
	if !slices.Equal(fixture.keyAPI.created, fixture.keyAPI.hidden) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
	if len(fixture.binding.saved) != 0 {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
	var classified *runtime.LatheError
	if !errors.As(err, &classified) {
		t.Fatalf("error was not classified: %v", err)
	}
	if strings.Contains(classified.Message, "duplicate_key_name") {
		t.Fatalf("raw error code reached the user: %q", classified.Message)
	}
	full := classified.Message + " " + classified.Hint
	for _, want := range []string{
		canonicalTestKeyName(),
		"already exists in this organization",
		"tokener agent key regenerate",
		"tokener keys revoke",
		"tokener agent key login",
	} {
		if !strings.Contains(full, want) {
			t.Fatalf("missing %q in %q", want, full)
		}
	}
}

// TestLoginClaimsASuffixedKeyFromAnEarlierInterruptedRun covers a login that was
// interrupted after it had already fallen back to a numbered name: the retry has
// to adopt that key rather than step over it and take a third name.
func TestLoginClaimsASuffixedKeyFromAnEarlierInterruptedRun(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.exists = false
	// The canonical name is held by a key this caller cannot see; the first
	// numbered name is the one this machine's interrupted run created.
	fixture.keyAPI.hidden = []string{testIdentity.keyName(1)}
	fixture.keyAPI.seed("key-suffixed", testIdentity.keyName(2), "sk-suffixed")

	if err := fixture.execute("key", "login"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.revealed, []string{"key-suffixed"}) {
		t.Fatalf("revealed = %q", fixture.keyAPI.revealed)
	}
	// The third name must never be reached, and nothing new may be created.
	if !slices.Equal(fixture.keyAPI.created, []string{testIdentity.keyName(1), testIdentity.keyName(2)}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
	if len(fixture.binding.saved) != 1 || fixture.binding.saved[0].Key != "sk-suffixed" {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
	if name := fixture.binding.saved[0].Name; name != testIdentity.keyName(2) {
		t.Fatalf("binding name = %q", name)
	}
}

func TestRegenerateRevokesThenCreatesAndRebinds(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.binding = agentBinding{
		Key:       "sk-old",
		KeyID:     "key-old",
		Name:      canonicalTestKeyName(),
		MachineID: testIdentity.Digest,
	}
	fixture.binding.exists = true
	fixture.keyAPI.seed("key-old", canonicalTestKeyName(), "sk-old")

	if err := fixture.execute("key", "regenerate"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.revoked, []string{"key-old"}) {
		t.Fatalf("revoked = %q", fixture.keyAPI.revoked)
	}
	// The revoke frees the name, so rotation reuses it rather than suffixing.
	if !slices.Equal(fixture.keyAPI.created, []string{canonicalTestKeyName()}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
	// Rotation must not adopt the key it just revoked.
	if len(fixture.keyAPI.revealed) != 0 {
		t.Fatalf("revealed = %q", fixture.keyAPI.revealed)
	}
	if len(fixture.binding.saved) != 1 {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
	stored := fixture.binding.saved[0]
	if stored.KeyID == "key-old" || stored.Key == "sk-old" || stored.Name != canonicalTestKeyName() {
		t.Fatalf("binding was not rotated = %#v", stored)
	}
}

func TestRegenerateLocatesKeyByNameWhenBindingHasNoID(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.binding = agentBinding{Key: "sk-old"}
	fixture.binding.exists = true
	fixture.keyAPI.seed("key-by-name", canonicalTestKeyName(), "sk-old")

	if err := fixture.execute("key", "regenerate"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.revoked, []string{"key-by-name"}) {
		t.Fatalf("revoked = %q", fixture.keyAPI.revoked)
	}
}

func TestRegenerateLocatesLegacyKeyByItsOriginalName(t *testing.T) {
	fixture := newAgentFixture(t)
	// A binding written before per-machine names carries no id and no name, so
	// the key is found under the single name every key used to have.
	fixture.binding.binding = agentBinding{Key: "sk-legacy-secret"}
	fixture.binding.exists = true
	fixture.keyAPI.seed("key-legacy", agentKeyNamePrefix, "sk-legacy-secret")

	if err := fixture.execute("key", "regenerate"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.revoked, []string{"key-legacy"}) {
		t.Fatalf("legacy key was not rotated: revoked = %q", fixture.keyAPI.revoked)
	}
	if !slices.Equal(fixture.keyAPI.created, []string{canonicalTestKeyName()}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
}

// TestRegenerateAfterManualRevokeCompletes covers the workaround from the
// issue: a user who revoked the key by hand must still be able to rotate.
func TestRegenerateAfterManualRevokeCompletes(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.binding = agentBinding{Key: "sk-old", KeyID: "key-old", Name: canonicalTestKeyName()}
	fixture.binding.exists = true
	fixture.keyAPI.revokeErr = alreadyRevokedResponse()

	if err := fixture.execute("key", "regenerate"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture.keyAPI.created, []string{canonicalTestKeyName()}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
	if len(fixture.binding.saved) != 1 || fixture.binding.saved[0].KeyID != "key-1" {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
}

func TestRegenerateReportsHowToRecoverWhenCreateFailsAfterRevoke(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.binding = agentBinding{Key: "sk-old", KeyID: "key-old", Name: canonicalTestKeyName()}
	fixture.binding.exists = true
	// Every name is taken by keys this caller cannot see, so the create fails
	// after the revoke has already taken the old key out of service.
	fixture.keyAPI.hidden = []string{
		testIdentity.keyName(1),
		testIdentity.keyName(2),
		testIdentity.keyName(3),
	}

	err := fixture.execute("key", "regenerate")
	if err == nil {
		t.Fatal("failed rotation was reported as success")
	}
	if !slices.Equal(fixture.keyAPI.revoked, []string{"key-old"}) {
		t.Fatalf("revoked = %q", fixture.keyAPI.revoked)
	}
	if len(fixture.binding.saved) != 0 {
		t.Fatalf("binding = %#v", fixture.binding.saved)
	}
	var classified *runtime.LatheError
	if !errors.As(err, &classified) {
		t.Fatalf("error was not classified: %v", err)
	}
	// The stale binding blocks `key login`, so the hint must not suggest it.
	if !strings.Contains(classified.Hint, "tokener agent key regenerate") {
		t.Fatalf("hint = %q", classified.Hint)
	}
	if !strings.Contains(classified.Message, "was revoked") || !strings.Contains(classified.Message, canonicalTestKeyName()) {
		t.Fatalf("message = %q", classified.Message)
	}
}

func alreadyRevokedResponse() error {
	return &runtime.HTTPError{
		Status:      http.StatusBadRequest,
		ContentType: "application/json",
		Body:        []byte(`{"error":"api_key_revoked"}`),
	}
}

func TestRegenerateWithoutAnyExistingKeyStillBinds(t *testing.T) {
	fixture := newAgentFixture(t)
	fixture.binding.exists = false

	if err := fixture.execute("key", "regenerate"); err != nil {
		t.Fatal(err)
	}
	if len(fixture.keyAPI.revoked) != 0 {
		t.Fatalf("revoked = %q", fixture.keyAPI.revoked)
	}
	if !slices.Equal(fixture.keyAPI.created, []string{canonicalTestKeyName()}) {
		t.Fatalf("created = %q", fixture.keyAPI.created)
	}
}

// TestLoginOnSecondMachineSucceedsAgainstOneOrganization reproduces issue #37:
// two machines binding against the same organization, the second having no
// local binding.
func TestLoginOnSecondMachineSucceedsAgainstOneOrganization(t *testing.T) {
	organization := &fakeKeys{secret: "sk"}
	machines := []machineIdentity{
		{Hostname: "macbook", Digest: "aaa111"},
		{Hostname: "macbook", Digest: "bbb222"},
	}
	names := map[string]bool{}
	for index, identity := range machines {
		fixture := newAgentFixture(t)
		fixture.binding.exists = false
		fixture.dependencies.keys = organization
		fixture.dependencies.identity = func() (machineIdentity, error) { return identity, nil }
		if err := fixture.execute("key", "login"); err != nil {
			t.Fatalf("machine %d: %v", index, err)
		}
		if len(fixture.binding.saved) != 1 {
			t.Fatalf("machine %d binding = %#v", index, fixture.binding.saved)
		}
		names[fixture.binding.saved[0].Name] = true
	}
	if len(names) != 2 {
		t.Fatalf("machines sharing a hostname collided on %v", slices.Collect(maps.Keys(names)))
	}
	if len(organization.created) != 2 {
		t.Fatalf("created = %q", organization.created)
	}
}
