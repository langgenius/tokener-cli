//go:build darwin && amd64

package agent

import _ "embed"

//go:embed assets/rx-darwin-amd64
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.7"
	embeddedRXSHA256  = "468cdb7395a9b18f7450bb23a7389759c454affcdd0e79633a633ad168cfcad3"
	embeddedRXOS      = "darwin"
	embeddedRXArch    = "amd64"
)
