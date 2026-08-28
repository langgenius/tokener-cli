//go:build windows && amd64

package agent

import _ "embed"

//go:embed assets/rx-windows-amd64.exe
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.7"
	embeddedRXSHA256  = "bd150b2629ddb0c78af9bd04fba21f2310b347889962b67cf4fe81d1cbcdb97f"
	embeddedRXOS      = "windows"
	embeddedRXArch    = "amd64"
)
