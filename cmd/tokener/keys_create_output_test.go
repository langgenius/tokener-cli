package main

import (
	"strings"
	"testing"

	"github.com/lathe-cli/lathe/pkg/runtime"

	"github.com/langgenius/tokener-cli/internal/generated/console"
)

func TestKeysCreateDefaultTableOmitsSecret(t *testing.T) {
	var spec *runtime.CommandSpec
	for i := range console.Specs {
		candidate := &console.Specs[i]
		if candidate.Group == "keys" && candidate.Use == "create" {
			spec = candidate
			break
		}
	}
	if spec == nil {
		t.Fatal("keys create spec not found")
	}
	const secret = "sk-verysecretvalue1234567890"
	response := []byte(`{"key":"` + secret + `","record":{"id":"05e81be8-0eca-4ab9-b333-b6b51a6aa301","name":"demo","prefix":"sk-verys","status":"preparing","createdAt":"2026-08-30T00:00:00Z"}}`)
	var out strings.Builder
	if err := runtime.FormatOutput(response, "table", &out, spec.Output); err != nil {
		t.Fatalf("format table: %v", err)
	}
	rendered := out.String()
	if strings.Contains(rendered, secret) {
		t.Fatalf("default table output contains the plaintext secret:\n%s", rendered)
	}
	if !strings.Contains(rendered, "demo") {
		t.Fatalf("default table output lost the record columns:\n%s", rendered)
	}
}
