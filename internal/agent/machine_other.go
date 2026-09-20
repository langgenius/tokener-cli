//go:build !darwin && !linux && !windows

package agent

// platformMachineID has no stable source on this platform; the caller falls
// back to a generated identifier persisted in the CLI config.
func platformMachineID() (string, error) {
	return "", errNoPlatformMachineID
}
