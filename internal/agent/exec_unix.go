//go:build !windows

package agent

import "syscall"

func execProcess(path string, args, environment []string) error {
	return syscall.Exec(path, append([]string{path}, args...), environment)
}
