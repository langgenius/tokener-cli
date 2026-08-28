//go:build linux && amd64

package agent

import _ "embed"

//go:embed assets/rx-linux-amd64
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.6"
	embeddedRXSHA256  = "258765258546db6cb268b17fe86e47c6fdab042786faeafe4ec9e1d0bd865008"
	embeddedRXOS      = "linux"
	embeddedRXArch    = "amd64"
)
