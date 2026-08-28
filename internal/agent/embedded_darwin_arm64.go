//go:build darwin && arm64

package agent

import _ "embed"

//go:embed assets/rx
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.6"
	embeddedRXSHA256  = "22e344d72afb2827ce3613ab2c7db35957261c0ea19ca171d2aa8591b765067a"
	embeddedRXOS      = "darwin"
	embeddedRXArch    = "arm64"
)
