LATHE_MODULE ?= github.com/lathe-cli/lathe
LATHE_VERSION ?= $(shell go list -m -f '{{.Version}}' $(LATHE_MODULE))
LATHE_REF ?= latest
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: cli-sync cli-build cli-install rx-verify rx-update lathe-update test check ci-check release-snapshot

cli-sync:
	cp cli.yaml cmd/tokener/cli.yaml
	go run github.com/lathe-cli/lathe/cmd/lathe@$(LATHE_VERSION) bootstrap -overlay overlays
	go mod tidy

cli-build:
	go build -ldflags "-X github.com/lathe-cli/lathe/pkg/lathe.Version=$$(git describe --tags --always 2>/dev/null || echo dev)" -o bin/tokener ./cmd/tokener

cli-install: cli-build
	mkdir -p "$(INSTALL_DIR)"
	ln -sfn "$(CURDIR)/bin/tokener" "$(INSTALL_DIR)/tokener"

rx-verify:
	go run ./internal/cmd/rxmanifest verify

rx-update:
	@test -n "$(RX_TAG)" || { echo "RX_TAG is required: make rx-update RX_TAG=v0.6.1"; exit 1; }
	go run ./internal/cmd/rxmanifest pull -tag "$(RX_TAG)"
	go test ./internal/agent/...

lathe-update:
	go get $(LATHE_MODULE)@$(LATHE_REF)
	go mod tidy
	$(MAKE) cli-sync

test: rx-verify cli-build
	go test ./...

check: cli-sync test
	go vet ./...

ci-check: rx-verify
	go build -o bin/tokener ./cmd/tokener
	go test ./...
	go vet ./...

release-snapshot: rx-verify
	goreleaser release --snapshot --clean
