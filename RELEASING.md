# Releases

Tokener is an L1 CLI under the repository baseline. The Git tag is the version source.
GoReleaser injects the version through `lathe.Version`. Lathe uses this value first and `debug.ReadBuildInfo()` as its fallback.
Do not add a separate version file or change published tags.

## Prepare a release

Use `vMAJOR.MINOR.PATCH` for stable releases. Use `-alpha.N`, `-beta.N`, or `-rc.N` for prereleases.
Start `N` at 1. Do not use leading zeros or build metadata.
During `0.y.z`, increment MINOR for incompatible changes. Use an RC when packaging, configuration, or storage formats change.
Choose the next version from the changes and their compatibility requirements.

Install git-cliff 2.14.1, the version used by the release workflow.
Set `RELEASE_TAG` to the chosen tag. Before tagging, generate the changelog:

```sh
git-cliff --config cliff.toml --ignore-tags='.*-(alpha|beta|rc)\..*' --tag "$RELEASE_TAG" --output CHANGELOG.md
make ci-check
make lint
make release-snapshot
```

Review the changelog against the commits. Include `CHANGELOG.md` in the release preparation PR.
Use the `chore(release):` prefix for the preparation PR title so that commit does not add its own release note.
The changelog groups the existing Conventional Commits by release tag. It omits release and Homebrew update commits.
Each release includes all changes since the previous stable release. Prerelease tags do not split this range.
Headings omit dates so tag creation time cannot change the prepared release notes.

After the preparation PR merges and release approval is given, tag that commit and push the tag.
Release approval is separate from PR approval. Do not tag a commit whose changelog lacks the matching release section.

## Release checks

The first release step rejects invalid tags and records stable or prerelease status.
The workflow then runs git-cliff and compares the tagged release section with the matching `CHANGELOG.md` section.
Missing, duplicate, empty, or changed sections stop the release before builds.
GoReleaser receives the verified section as its release notes.
The workflow runs `make ci-check` and checks for tracked file changes before publishing.

GoReleaser marks prereleases with `release.prerelease: auto`.
Homebrew uses `skip_upload: auto`, so prereleases do not open stable formula update PRs.
Stable releases retain the existing `Formula/tokener.rb` update path in this repository.

To build without publishing, run `make release-snapshot`.
