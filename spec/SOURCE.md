# Spec provenance

`spec/v1.yaml` is a **vendored copy** of the Tokener.ai Console Management API
contract. Do not edit it by hand.

| Field | Value |
| --- | --- |
| Upstream repository | `langgenius/ai-gateway` (private) |
| Upstream path | `apps/console/openapi/v1.yaml` |
| Upstream commit | `cd8d8c44f7fc5ad20ee1d90a68cddc11b649df2b` |
| Copied from | `git show HEAD:apps/console/openapi/v1.yaml` (committed content only) |
| SHA-256 | `4085c3cad9446c7ca4481fe4de71edd4e859a1e3b7e084076f1ff31a0df8a8f4` |

## Why it is vendored

The upstream repository is private, so `specs/sources.yaml` cannot clone it with
`repo_url`. The spec is checked in here instead and referenced from
`specs/sources.yaml` with `local_path: ../spec` (relative to that file).

## How to update

Update this file only through an upstream sync pull request:

1. Copy the committed upstream spec:
   `git -C <ai-gateway> show <sha>:apps/console/openapi/v1.yaml > spec/v1.yaml`
2. Refresh the commit SHA and SHA-256 in the table above.
3. Run `make cli-sync` and commit the regenerated `internal/generated/`,
   `skills/tokener/`, and `cmd/tokener/cli.yaml` output with the spec change.

Never apply local edits to `spec/v1.yaml`; the upstream generator
(`pnpm --filter console openapi:gen`) owns its content.
