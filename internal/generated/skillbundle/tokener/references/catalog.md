# Command discovery

1. Run `tokener search "<intent>" --json` for candidates, then `tokener commands show <path...> --json` before execution. Use `tokener commands --json` for the full catalog; add `--include-hidden` only when relevant.
2. Read the command's `http`, `flags`, `body`, `auth`, `mutation`, `dry_run`, `output`, `notes`, and `known_errors`. Do not infer write safety from the HTTP method or a flag name.
3. If `auth.required=true`, inspect `tokener auth status -o json`. Stop for login if that host has no credentials. Resolution order: `--hostname`, `$TOKENER_HOST`, selected host, command default host, then sole stored host. Multiple stored hosts may produce a current-host notice on stderr.
4. For a mutation, use the declared `http_preview` dry run before executing unless the user confirmed execution. If preview is unsupported, obtain explicit confirmation. A dry run does not fetch a `body.runtime_schema`; execution does.

## Inputs and results

- Use only declared flags. For sensitive flags, prefer their advertised `env`, `file`, or `stdin` input mode; select exactly one mode per flag.
- Pass JSON using `--file path` or `--file -`. `--set key.path=value` infers JSON types; `--set-str key.path=value` preserves strings, including decimal money values.
- Prefer `-o json`. On nonzero exit read `error.code`, `error.message`, and `error.hint`; optional `error.http` contains the status, not a response body. Human output also supports table, YAML, and raw.
- Before building durable catalog tooling, read `tokener commands schema --json` for the current schema and preview-result contract. Command details and schema are the authority for supported fields and behavior.
