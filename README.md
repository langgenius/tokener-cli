# tokener-cli

`tokener` is the command-line client for the Tokener.ai Console Management API.
It authenticates with a Tokener.ai management personal access token (PAT,
prefix `tkr_pat_`) and exposes the console management surface — API keys,
billing balance and ledger, usage, and the public model catalog.

It is not the LLM data plane: model traffic goes to the Tokener Gateway with a
Tokener.ai API key, not to this CLI.

## Architecture

```mermaid
flowchart LR
  spec["Pinned OpenAPI + overlays"] -->|Lathe| generated["internal/generated"]
  generated --> main["cmd/tokener"]
  main -->|API commands / Lathe auth| console["Console management API"]
  main --> command["agent/command.go"]
  command --> keys["agent/keys.go"]
  keys -->|create key| console
  keys --> binding["agent/binding.go: per-host key files"]
  command --> engine["agent/engine.go: verified rx cache"]
  engine --> launch["agent/launch.go: host protocol"]
  launch --> harness["Native coding harness"]
  harness -->|model traffic| gateway["Tokener Gateway"]
  lock["rx.lock.json + four native assets"] --> engine
```

`host.go` resolves the management host and its gateway. Agent keys stay in
per-host config files; only the default host reads the legacy `agent-key.json`.
`machine.go` derives a per-machine key name, `Tokener Agent CLI · <host>-<id>`,
where the id is a six-character digest of the platform machine identifier; only
the digest is sent. Because key names are unique per organization, that name is
what lets one organization hold one key per machine. `key login` claims the key
carrying this machine's name when it already exists and creates one otherwise,
so it is safe to rerun; `key regenerate` revokes that key and issues a new one.
`atomicfile` owns temporary-file writes and platform-specific replacement for
bindings, cached engines, and the snapshot lock. Engine extraction verifies
SHA-256, retains old digest directories for rollback, and validates `TOKENER_RX`
overrides through the rx host handshake. The launch request contains the gateway
profile; its credential travels separately in `TOKENER_API_KEY`.

`make cli-sync` rebuilds the management commands and bundled Skill from the
pinned spec, overlay, and `internal/skill-include/`. The Skill discovers commands
through the binary's catalog. `refresh-rx.yml` builds and probes four native
engines, records their source and checksums, and opens a snapshot PR. Release
tags run `make ci-check` before GoReleaser packages the binaries and opens the
Homebrew update PR.

## Install

Install the latest release on macOS or Linux:

```sh
curl -fsSL https://tokener.dev/install.sh | sh
```

Install with Homebrew:

```sh
brew tap langgenius/tokener https://github.com/langgenius/tokener-cli
brew install langgenius/tokener/tokener
```

Upgrade an existing installation with `tokener update` or `brew upgrade
tokener`, depending on how it was installed. Windows archives are available on
the [GitHub Releases](https://github.com/langgenius/tokener-cli/releases) page.

## Layout

| Path | Owner |
| --- | --- |
| `specs/sources.yaml` | spec source declaration for `lathe specsync`, pinned to an upstream tag |
| `cli.yaml` | generated CLI identity, auth validation, skill, and update config |
| `overlays/console.yaml` | human-facing command names, help, examples, and parameter presentation |
| `cmd/tokener/main.go` | runtime entrypoint and auth-login defaults |
| `internal/agent/` | key lifecycle, host selection, engine extraction, and launch |
| `internal/rxsnapshot/` | embedded engine provenance and artifact verification |
| `internal/skill-include/` | authored Skill instructions and catalog protocol |
| `internal/generated/` | generated command specs (do not edit) |
| `skills/tokener/` | generated agent Skill (do not edit) |

## Build

```sh
make cli-sync    # regenerate from spec/ and cli.yaml, then go mod tidy
make cli-build   # build bin/tokener
make check       # cli-sync + tests + go vet
make release-snapshot # build release artifacts without publishing
```

`make cli-sync` runs the pinned `lathe` generator via `go run`, reading the
version from `go.mod`; override it with `make cli-sync LATHE_VERSION=vX.Y.Z`.

## Dependency updates

```sh
make rx-update RX_TAG=v0.6.1   # repin the embedded engines to a Recall release
make lathe-update              # move lathe to its latest version and regenerate
make lathe-update LATHE_REF=v0.6.2
```

`make rx-update` downloads the four `recall-*` archives of that release, keeps
only their `rx` member, rewrites `internal/agent/rx.lock.json` with the tag's
commit and the new checksums, and verifies the result. All four engines come
from one release build, so every platform ships the same revision; commit the
assets and the lock together. `refresh-rx.yml` stays the path for pinning an
unreleased Recall commit, building the four engines itself and opening the
snapshot PR.

`make lathe-update` moves the pin in `go.mod`, the single source for
`LATHE_VERSION`, then re-runs `make cli-sync` so the generated output matches
the new generator.

## Generated output

`internal/generated/`, `skills/tokener/`, and `cmd/tokener/cli.yaml` are
generated by `lathe`. Do not hand-edit them. To change CLI behavior, change
`cli.yaml`, `specs/sources.yaml`, `overlays/console.yaml`, or the upstream spec,
then re-run `make cli-sync` and commit the regenerated output with the change.

## Usage

```sh
bin/tokener auth login
bin/tokener auth login --no-browser
bin/tokener auth login --with-token
bin/tokener commands --json
bin/tokener commands show keys create --json
bin/tokener keys list -o json
bin/tokener billing ledger --all -o json
```
