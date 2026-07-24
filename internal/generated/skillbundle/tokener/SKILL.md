---
name: tokener
description: >
  Use when operating the tokener generated CLI. Discover commands, inspect parameters,
  check auth state, and execute API operations safely.
---

# tokener CLI

Use this skill when a user asks you to operate `tokener`, inspect its API commands, or find the right generated command for an API task.

## Workflow

1. Search for candidates with `tokener search "<intent>" --json`; use `--limit` when needed. Search is only candidate discovery.
2. Inspect the exact command with `tokener commands show <path...> --json` before executing an unfamiliar command.
3. If the command detail has `auth.required=true`, run `tokener auth status --hostname <host>` before execution. Use `http.default_hostname` when present unless the user provides `--hostname` or `$TOKENER_HOST`.
4. Execute only after flags, body, auth, HTTP path, and output hints are clear from `commands show`.

## General Commands

- `tokener commands --json`: full generated command catalog.
- `tokener commands --include-hidden --json`: include hidden generated commands.
- `tokener commands show <path...> --json`: source of truth for one command.
- `tokener commands schema --json`: catalog schema version for parser compatibility.
- `tokener search "<intent>" --json`: ranked candidate commands.

## Maintenance Commands

- `tokener --version` or `tokener -v`: print CLI build version.
- `tokener update`: update this CLI from configured GitHub Releases. Run only when the user explicitly asks to update `tokener`; it may replace the current executable. Use `--yes` only when explicitly authorized.

## References

- Read `references/catalog.md` for the command discovery protocol and catalog field meanings.
- Read `references/modules/tokener-ai-console-management-api.md` for the `Tokener.ai Console Management API` module command index.

## Rules

- Do not guess flags or request body shape from command names.
- Do not execute directly from search results; confirm with `commands show` first.
- Prefer `-o json` for machine-readable command output unless the user asks for human-readable output.
- Use `--file`, `--set`, or `--set-str` for JSON request bodies according to `commands show` body requirements.
- For sensitive flags, prefer safe modes from `flags[].input_modes`: `--<flag>-env`, `--<flag>-file`, or `--<flag>-stdin`.
