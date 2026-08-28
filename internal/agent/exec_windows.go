//go:build windows

package agent

import (
	"errors"
	"os"
	"os/exec"
)

func execProcess(path string, args, environment []string) error {
	command := exec.Command(path, args...)
	command.Env = environment
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.ExitCode())
		}
		return err
	}
	return nil
}
