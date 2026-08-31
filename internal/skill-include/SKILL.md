## CLI-native commands

`tokener search` only indexes generated API commands. It does not find auth, agent, skill, update, or completion. If search returns no candidates, read `references/modules/` and this section.

- `tokener auth login`: use this exact command for Tokener browser device approval; no hostname or provider flag is needed. Use `--no-browser` to print the verification URL and code without opening a browser. Tokener stores the resulting personal access token and does not use OAuth access or refresh tokens.
- `tokener auth login --with-token`: authenticate from stdin with an existing personal access token whose value starts with `tkr_pat_`.
- `tokener auth status` and `tokener auth use <host>`: inspect or select the management host.
- `tokener agent [<harness>]`: launch a coding agent through the Tokener Gateway. Omit the harness in an interactive terminal to choose from the picker; scripts must pass one. Available harnesses: claude, codex, opencode, pi, dsh, kimi.
- `tokener agent key login` and `tokener agent key status`: bind or inspect the local agent key. Status is local-only and does not create or rotate a key.
- `tokener skill install`: install this skill into local agent hosts. Preview with `--dry-run` before `--yes`.
- `tokener update`: install a newer GitHub Release of this CLI. Do not pass `--yes` unless the user explicitly asked to replace the binary.
- `tokener completion`: generate shell completion scripts.

### Write-path notes

- `tokener keys create`: set the name with `--name <name>`; the name is optional and the server generates one when omitted. Nested limits fields have no typed flags; set exact decimal budgets with `--set-str`, for example `--set-str limits.maxBudgetUsd=100`. The plaintext secret appears only in the `key` field of structured output and in `tokener keys reveal`.
- `tokener keys replace-limits` is a full replacement: omitted fields are cleared.
- Treat `keys create` and `keys reveal` output as credentials. Do not write them to logs or shell history.
