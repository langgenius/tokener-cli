//go:build windows

package agent

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

// platformMachineID reads MachineGuid, which Windows writes once at install time.
func platformMachineID() (string, error) {
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Cryptography`,
		registry.QUERY_VALUE|registry.WOW64_64KEY,
	)
	if err != nil {
		return "", errNoPlatformMachineID
	}
	defer func() { _ = key.Close() }()
	value, _, err := key.GetStringValue("MachineGuid")
	if err != nil {
		return "", errNoPlatformMachineID
	}
	if value = strings.TrimSpace(value); value == "" {
		return "", errNoPlatformMachineID
	}
	return value, nil
}
