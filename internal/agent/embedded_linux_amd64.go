//go:build linux && amd64

package agent

import _ "embed"

//go:embed assets/rx-linux-amd64
var embeddedRX []byte

const (
	embeddedRXOS   = "linux"
	embeddedRXArch = "amd64"
)
