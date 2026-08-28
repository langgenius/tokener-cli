//go:build windows && amd64

package agent

import _ "embed"

//go:embed assets/rx-windows-amd64.exe
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.6"
	embeddedRXSHA256  = "18d94f81937d26fcdabbf4d0b4d54654277bed5241f70d46c28a38f6bbad0f02"
	embeddedRXOS      = "windows"
	embeddedRXArch    = "amd64"
)
