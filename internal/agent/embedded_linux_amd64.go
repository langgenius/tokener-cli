//go:build linux && amd64

package agent

import _ "embed"

//go:embed assets/rx-linux-amd64
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.7"
	embeddedRXSHA256  = "e168b4d6d4edc33a1f343d4a905c1b3ef2d8427d72142eb517a9caae052b458e"
	embeddedRXOS      = "linux"
	embeddedRXArch    = "amd64"
)
