LATHE_VERSION ?= v0.4.5

.PHONY: cli-sync cli-build test check

cli-sync:
	cp cli.yaml cmd/tokener/cli.yaml
	go run github.com/lathe-cli/lathe/cmd/lathe@$(LATHE_VERSION) bootstrap
	go mod tidy

cli-build:
	go build -o bin/tokener ./cmd/tokener

test: cli-build
	go test ./...

check: cli-sync test
	go vet ./...
