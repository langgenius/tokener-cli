package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFormatKeyNameStaysWithinTheServerLimit(t *testing.T) {
	pathological := strings.Repeat("very-long-machine-name", 20)
	for _, test := range []struct {
		name, hostname string
	}{
		{"short hostname", "box"},
		{"pathological hostname", pathological},
		{"multi-byte hostname", strings.Repeat("机器", 40)},
		{"empty hostname", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			for attempt := 1; attempt <= maxKeyNameAttempts; attempt++ {
				got := formatKeyName(test.hostname, "abc123", attempt)
				// The server measures characters, not bytes, and rejects >64.
				if length := utf8.RuneCountInString(got); length > maxKeyNameLength {
					t.Fatalf("attempt %d: %q is %d runes", attempt, got, length)
				}
				if !strings.HasPrefix(got, agentKeyNamePrefix+agentKeyNameSeparator) {
					t.Fatalf("attempt %d: %q lost its prefix", attempt, got)
				}
				if !strings.Contains(got, "abc123") {
					t.Fatalf("attempt %d: %q lost the machine digest", attempt, got)
				}
			}
		})
	}
	// The separator is multi-byte, so a byte-length check would have passed a
	// name the server rejects. Guard the assumption the rune cap protects.
	if utf8.RuneCountInString(agentKeyNameSeparator) == len(agentKeyNameSeparator) {
		t.Fatal("separator is no longer multi-byte; the rune cap test is vacuous")
	}
}

func TestFormatKeyNameDistinguishesMachinesAndAttempts(t *testing.T) {
	shared := "macbook-pro"
	first := formatKeyName(shared, "aaa111", 1)
	second := formatKeyName(shared, "bbb222", 1)
	if first == second {
		t.Fatalf("machines sharing a hostname produced the same name %q", first)
	}
	if got := formatKeyName(shared, "aaa111", 1); got != first {
		t.Fatalf("name is not stable: %q then %q", first, got)
	}
	if suffixed := formatKeyName(shared, "aaa111", 2); suffixed == first || !strings.HasSuffix(suffixed, "-2") {
		t.Fatalf("attempt 2 name = %q", suffixed)
	}
	// A truncated hostname must not erase the digest that keeps names distinct.
	long := strings.Repeat("x", 200)
	if formatKeyName(long, "aaa111", 1) == formatKeyName(long, "bbb222", 1) {
		t.Fatal("truncation collapsed two machines onto one name")
	}
}

func TestMachineIdentifierIsResolvedOnceThenCached(t *testing.T) {
	directory := bindAgentTestManifest(t)
	path, err := machineIDPath()
	if err != nil {
		t.Fatal(err)
	}

	first, err := machineIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	if first == "" {
		t.Fatal("identifier is empty")
	}
	second, err := machineIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("identifier changed from %q to %q", first, second)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != first {
		t.Fatalf("persisted %q, returned %q", data, first)
	}
	if !strings.HasPrefix(path, directory) {
		t.Fatalf("identifier stored outside the CLI config: %q", path)
	}
	if mode := fileMode(t, path); mode&0o077 != 0 {
		t.Fatalf("identifier mode = %o", mode)
	}
}

// TestMachineDigestPrefersTheCacheOverThePlatform guards the property that
// keeps this machine's key reachable: once an identifier is cached, a platform
// source that fails or reports something else cannot move the digest, and so
// cannot rename this machine and strand the key it already owns.
func TestMachineDigestPrefersTheCacheOverThePlatform(t *testing.T) {
	bindAgentTestManifest(t)
	path, err := machineIDPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := writeMachineID(path, "cached-identifier"); err != nil {
		t.Fatal(err)
	}

	identifier, err := machineIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	if identifier != "cached-identifier" {
		t.Fatalf("identifier = %q; the platform source displaced the cache", identifier)
	}
	digest, err := machineDigest()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("cached-identifier"))
	if want := hex.EncodeToString(sum[:])[:machineDigestLength]; digest != want {
		t.Fatalf("digest = %q, want %q", digest, want)
	}
}

func TestGeneratedMachineIDIsAUniqueUUIDv4(t *testing.T) {
	first, err := newUUID()
	if err != nil {
		t.Fatal(err)
	}
	// A UUIDv4: 36 characters, version 4, RFC 4122 variant.
	if len(first) != 36 || first[14] != '4' || !strings.ContainsRune("89ab", rune(first[19])) {
		t.Fatalf("identifier = %q", first)
	}
	second, err := newUUID()
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatalf("two generated identifiers are equal: %q", first)
	}
}

func TestMachineDigestIsShortStableAndNotTheRawIdentifier(t *testing.T) {
	bindAgentTestManifest(t)
	identifier, err := machineIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	digest, err := machineDigest()
	if err != nil {
		t.Fatal(err)
	}
	// The identifier itself differs by host, so assert only the properties that
	// hold wherever it came from.
	if len(digest) != machineDigestLength {
		t.Fatalf("digest = %q", digest)
	}
	if strings.Contains(identifier, digest) && len(identifier) > machineDigestLength {
		t.Fatalf("digest %q leaks the raw identifier %q", digest, identifier)
	}
	again, err := machineDigest()
	if err != nil || again != digest {
		t.Fatalf("digest changed from %q to %q (%v)", digest, again, err)
	}
}

func TestLocalHostnameFallsBackAndDropsDomainSuffix(t *testing.T) {
	if hostname := localHostname(); hostname == "" || strings.Contains(hostname, ".") {
		t.Fatalf("hostname = %q", hostname)
	}
}
