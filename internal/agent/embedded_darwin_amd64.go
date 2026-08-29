//go:build darwin && amd64

package agent

import _ "embed"

//go:embed assets/rx-darwin-amd64
var embeddedRX []byte

const (
	embeddedRXOS   = "darwin"
	embeddedRXArch = "amd64"
)
