//go:build darwin && arm64

package agent

import _ "embed"

//go:embed assets/rx
var embeddedRX []byte

const (
	embeddedRXVersion = "0.5.7"
	embeddedRXSHA256  = "59c15dc19142a44209f658d7f51905d9bd82afb8a67e4249af14130adca0ae39"
	embeddedRXOS      = "darwin"
	embeddedRXArch    = "arm64"
)
