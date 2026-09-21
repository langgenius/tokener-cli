# Contributing

## Before you open a PR

- Run `make ci-check` and `make lint` before pushing. If you changed the CLI definition, run `make check` instead of `make ci-check`; it regenerates output first (see [Build](README.md#build)). `make lint` needs [golangci-lint](https://golangci-lint.run/welcome/install/) v2.13 or newer. CI also validates the PR title and builds every release platform.
- Do not hand-edit generated output such as `internal/generated/`, `skills/tokener/`, and `cmd/tokener/cli.yaml` (see [Generated output](README.md#generated-output)).
- Keep one concern per PR. Anything else you notice goes in an issue.

## Code

- Write no comments in Go code. Put the reasoning in names, tests, and the PR description. Build constraints, `//go:` directives, `//nolint`, and generated files are exempt. CI rejects PRs that add comment lines to Go source.
- A new dependency needs a reason in the PR: what it does that the standard library and the existing dependencies do not.

## Commits and pull requests

- `main` is protected: no direct pushes. A PR needs green required checks and one approval before merge.
- Title the PR as a Conventional Commit, `type(scope): subject`. Types in use are `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, and `revert`; the scope is the area you touched, such as `agent`, `cli`, `keys`, or `rx`. CI checks the title. PRs are squash-merged with the title as the commit subject, and GitHub appends `(#123)`, so leave that out.
- Mark breaking changes with `!` after the type/scope, for example `feat(api)!: ...`, and explain the break in the PR description.
- Sign off your commits with `git commit -s`. Commit messages need no body; the PR description carries the context.
- Fill in every section of the PR template: what changed, why, how you verified it, and known gaps. For verification, list the commands you ran and what they printed; for user-visible behavior, show the real before and after. If a section does not apply, write `None` instead of deleting it.
