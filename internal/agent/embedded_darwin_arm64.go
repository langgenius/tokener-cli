//go:build darwin && arm64

package agent

import _ "embed"

//go:embed assets/rx
var embeddedRX []byte

const (
	embeddedRXOS   = "darwin"
	embeddedRXArch = "arm64"
)
