//go:build linux

package agent

import (
	"os"
	"strings"
)

// platformMachineID reads the systemd/D-Bus machine identifier.
func platformMachineID() (string, error) {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if value := strings.TrimSpace(string(data)); value != "" {
			return value, nil
		}
	}
	return "", errNoPlatformMachineID
}
