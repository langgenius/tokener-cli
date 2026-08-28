//go:build !windows

package agent

import "os"

func restrictDirectory(path string) error {
	return os.Chmod(path, 0o700)
}

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
