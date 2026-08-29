LATHE_VERSION ?= v0.6.1-0.20260829152256-9d4892b29474
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: cli-sync cli-build cli-install rx-verify test check ci-check release-snapshot

cli-sync:
	cp cli.yaml cmd/tokener/cli.yaml
	go run github.com/lathe-cli/lathe/cmd/lathe@$(LATHE_VERSION) bootstrap -overlay overlays
	go mod tidy

cli-build:
	go build -o bin/tokener ./cmd/tokener

cli-install: cli-build
	mkdir -p "$(INSTALL_DIR)"
	ln -sfn "$(CURDIR)/bin/tokener" "$(INSTALL_DIR)/tokener"

rx-verify:
	go run ./internal/cmd/rxmanifest verify

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
