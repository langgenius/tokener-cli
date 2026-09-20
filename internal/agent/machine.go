package agent

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/langgenius/tokener-cli/internal/atomicfile"
)

const (
	// agentKeyNamePrefix is the historical name of the agent key. It is kept as
	// the leading segment so keys stay recognisable in the console listing.
	agentKeyNamePrefix = "Tokener Agent CLI"
	// agentKeyNameSeparator joins the prefix and the per-machine segment.
	agentKeyNameSeparator = " · "
	// maxKeyNameLength mirrors MAX_NAME_LENGTH in the console key service. The
	// server measures the trimmed name in characters, so the CLI counts runes.
	maxKeyNameLength = 64
	// maxHostnameLength bounds the hostname segment so the digest always fits.
	maxHostnameLength = 30
	// machineDigestLength is how much of the machine identifier digest is sent.
	machineDigestLength = 6
	// machineIDFileName holds the cached identifier naming this machine.
	machineIDFileName = "machine-id"
)

// machineIdentity names one machine. Hostname is a human-readable label that
// machines may share; Digest is what makes the pair unique, and is the only
// part derived from the platform identifier.
type machineIdentity struct {
	Hostname string
	Digest   string
}

// keyName returns the key name for this machine. attempt is 1 for the canonical
// name; higher attempts append a disambiguating suffix, used when the canonical
// name is already taken in the organization.
func (identity machineIdentity) keyName(attempt int) string {
	return formatKeyName(identity.Hostname, identity.Digest, attempt)
}

// localMachineIdentity derives this machine's identity. Only the digest is ever
// sent to the server; the raw platform identifier never leaves the machine.
func localMachineIdentity() (machineIdentity, error) {
	digest, err := machineDigest()
	if err != nil {
		return machineIdentity{}, err
	}
	return machineIdentity{Hostname: localHostname(), Digest: digest}, nil
}

// formatKeyName builds "Tokener Agent CLI · <hostname>-<digest>" and keeps the
// whole name within the server limit by truncating only the hostname segment.
func formatKeyName(hostname, digest string, attempt int) string {
	suffix := ""
	if attempt > 1 {
		suffix = fmt.Sprintf("-%d", attempt)
	}
	identity := "-" + digest + suffix
	fixed := utf8.RuneCountInString(agentKeyNamePrefix + agentKeyNameSeparator + identity)
	budget := min(maxHostnameLength, maxKeyNameLength-fixed)
	if budget <= 0 {
		// Pathological: the digest alone fills the budget. Drop the hostname.
		return truncateRunes(agentKeyNamePrefix+agentKeyNameSeparator+digest+suffix, maxKeyNameLength)
	}
	return agentKeyNamePrefix + agentKeyNameSeparator + truncateRunes(hostname, budget) + identity
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

// localHostname returns a short, stable label for this machine. It is a display
// aid only: two machines may share it, which is why the digest is appended.
func localHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "host"
	}
	hostname = strings.TrimSpace(hostname)
	// Strip mDNS and DNS suffixes so the readable segment stays short.
	if label, _, found := strings.Cut(hostname, "."); found && label != "" {
		hostname = label
	}
	if hostname == "" {
		return "host"
	}
	return hostname
}

// machineDigest hashes the identifier naming this machine and returns the first
// machineDigestLength hex characters. Only the digest ever leaves the machine.
func machineDigest() (string, error) {
	identifier, err := machineIdentifier()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(sum[:])[:machineDigestLength], nil
}

// machineIdentifier returns the identifier naming this machine, reading the
// cached value before consulting the platform. The cache is what keeps the key
// name stable: a platform source that fails transiently would otherwise fall
// through to a generated identifier, rename this machine, and strand the key it
// already owns under the old name. The cache is seeded from the platform
// source, or from a generated UUIDv4 where the platform exposes none.
func machineIdentifier() (string, error) {
	path, err := machineIDPath()
	if err != nil {
		return "", err
	}
	identifier, cached, err := readMachineID(path)
	if err != nil {
		return "", err
	}
	if cached {
		return identifier, nil
	}
	identifier, err = platformMachineID()
	if identifier = strings.TrimSpace(identifier); err != nil || identifier == "" {
		if identifier, err = newUUID(); err != nil {
			return "", err
		}
	}
	if err := writeMachineID(path, identifier); err != nil {
		return "", err
	}
	return identifier, nil
}

func readMachineID(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read machine identifier: %w", err)
	}
	identifier := strings.TrimSpace(string(data))
	return identifier, identifier != "", nil
}

func writeMachineID(path, identifier string) error {
	if err := atomicfile.PrivateDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create agent config directory: %w", err)
	}
	if err := atomicfile.Write(path, []byte(identifier+"\n"), 0o600); err != nil {
		return fmt.Errorf("write machine identifier: %w", err)
	}
	return nil
}

func machineIDPath() (string, error) {
	dir, err := configDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, machineIDFileName), nil
}

func newUUID() (string, error) {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "", fmt.Errorf("generate machine identifier: %w", err)
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(buffer[:])
	return strings.Join([]string{encoded[:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:]}, "-"), nil
}

// errNoPlatformMachineID reports that this platform exposes no stable identifier.
var errNoPlatformMachineID = errors.New("no platform machine identifier")
