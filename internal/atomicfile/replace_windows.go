//go:build windows

package atomicfile

import (
	"os"

	"golang.org/x/sys/windows"
)

func replace(source, destination string) error {
	sourcePointer, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPointer, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		return os.Rename(source, destination)
	}
	return windows.MoveFileEx(sourcePointer, destinationPointer, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
