---
name: tokener
description: >
  Use when operating the tokener CLI: Tokener.ai console management
  (keys, billing, usage, models), authentication, or launching a coding
  agent through the Tokener Gateway.
---

# tokener

`tokener search "<intent>" --json` indexes API commands only. Inspect an unfamiliar API command with `tokener commands show <path...> --json` before running it. Do not guess flags or body shape, and do not execute from search results. Prefer `-o json`. On error, read `error.code`, `error.message`, and `error.hint`. If `mutation` is not `read`, pass `--dry-run` first unless the user confirmed execution.

API command index: `references/modules/tokener-ai-console-management-api.md`.

## Auth

- `tokener auth login`: browser device login. Defaults to `console.tokener.dev`. Use `--hostname` or `$TOKENER_HOST` only for another host. `--no-browser` prints the URL and code.
- `tokener auth login --with-token`: PAT on stdin (`tkr_pat_...`).
- `tokener auth status -o json` and `tokener auth use <host>`: inspect or select the host.

## Agent

- `tokener agent [<harness>]`: launch through the Tokener Gateway. Interactive terminals may omit the harness; scripts must pass one of: claude, codex, opencode, pi, dsh, kimi.
- `tokener agent key login` and `tokener agent key status`: bind or inspect the local agent key. Status does not create or rotate a key.

## Keys

- `tokener keys create`: `--name` is optional. Nested `limits` have no typed flags; set decimals with `--set-str`, for example `--set-str limits.maxBudgetUsd=100`.
- `tokener keys replace-limits` replaces the entire limit set; omitted fields are cleared.
- Treat `keys create` and `keys reveal` output as credentials. Do not write them to logs or shell history.

## Skill and update

- `tokener skill install`: preview with `--dry-run` before `--yes`.
- `tokener update`: do not pass `--yes` unless the user asked to replace the binary.
