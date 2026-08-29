//go:build !((darwin && (arm64 || amd64)) || (linux && amd64) || (windows && amd64))

package agent

var embeddedRX []byte

const (
	embeddedRXOS   = ""
	embeddedRXArch = ""
)
