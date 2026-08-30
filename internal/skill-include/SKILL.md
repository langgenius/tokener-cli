## CLI-native commands

`tokener search` only indexes generated API commands. It does not find auth, agent, skill, update, or completion. If search returns no candidates, read `references/modules/` and this section.

- `tokener auth login --with-token`: authenticate with a personal access token whose value starts with `tkr_pat_`.
- `tokener auth status` and `tokener auth use <host>`: inspect or select the management host.
- `tokener agent <harness>`: launch a coding agent through the Tokener Gateway. Available harnesses: claude, codex, opencode, pi, dsh, kimi.
- `tokener agent key login` and `tokener agent key status`: bind or inspect the local agent key. Status is local-only and does not create or rotate a key.
- `tokener skill install`: install this skill into local agent hosts. Preview with `--dry-run` before `--yes`.
- `tokener update`: install a newer GitHub Release of this CLI. Do not pass `--yes` unless the user explicitly asked to replace the binary.
- `tokener completion`: generate shell completion scripts.

### Write-path notes

- `tokener keys create` takes a JSON body. Set the name with `--set-str name=<name>`. The plaintext secret appears only in the JSON field `key` and in `tokener keys reveal`.
- `tokener keys replace-limits` is a full replacement: omitted fields are cleared.
- Treat `keys create` and `keys reveal` output as credentials. Do not write them to logs or shell history.
