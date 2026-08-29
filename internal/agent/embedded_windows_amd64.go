//go:build windows && amd64

package agent

import _ "embed"

//go:embed assets/rx-windows-amd64.exe
var embeddedRX []byte

const (
	embeddedRXOS   = "windows"
	embeddedRXArch = "amd64"
)
