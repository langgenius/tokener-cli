LATHE_VERSION ?= v0.6.0
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: cli-sync cli-build test check

cli-sync:
	cp cli.yaml cmd/tokener/cli.yaml
	go run github.com/lathe-cli/lathe/cmd/lathe@$(LATHE_VERSION) bootstrap
	go mod tidy

cli-build:
	go build -o bin/tokener ./cmd/tokener

cli-install: cli-build
	mkdir -p "$(INSTALL_DIR)"
	ln -sfn "$(CURDIR)/bin/tokener" "$(INSTALL_DIR)/tokener"

test: cli-build
	go test ./...

check: cli-sync test
	go vet ./...
