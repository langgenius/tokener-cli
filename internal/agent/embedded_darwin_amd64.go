//go:build darwin && amd64

package agent

import _ "embed"

//go:embed assets/rx-darwin-amd64
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.6"
	embeddedRXSHA256  = "0b0d7f542690e70fa1a8249c7ad5660998b91485e94b4e326a6c2eabafc7d7b0"
	embeddedRXOS      = "darwin"
	embeddedRXArch    = "amd64"
)
