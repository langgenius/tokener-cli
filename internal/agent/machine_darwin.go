//go:build darwin

package agent

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// platformMachineID reads IOPlatformUUID, the identifier macOS keeps stable for
// the lifetime of the machine.
func platformMachineID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return "", errNoPlatformMachineID
	}
	for line := range strings.Lines(string(output)) {
		_, value, found := strings.Cut(line, "\"IOPlatformUUID\"")
		if !found {
			continue
		}
		_, value, found = strings.Cut(value, "=")
		if !found {
			continue
		}
		if value = strings.Trim(strings.TrimSpace(value), "\""); value != "" {
			return value, nil
		}
	}
	return "", errNoPlatformMachineID
}
