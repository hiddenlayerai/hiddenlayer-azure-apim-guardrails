# Branch Protection for `main`

Configure these settings in GitHub UI for the `main` branch.

## Required Pull Request Reviews

- Require a pull request before merging.
- Require approvals: `1`.
- Require review from Code Owners: optional.
- Dismiss stale pull request approvals when new commits are pushed: optional but recommended.
- Require approval of the most recent reviewable push: recommended.

## Required Status Checks

Enable required checks and select:

- `Test`
- `Build`
- `Gate`
- `macOS pkg` (the `macos-package` job)

`Test` and `Build` are produced by `.github/workflows/cli-ci.yml`. `Gate` and
`macOS pkg` are produced by `.github/workflows/packaging-ci.yml`. `Gate`
always reports a status (deciding whether the packaging job runs), so
requiring it never deadlocks PRs that do not touch packaging paths.
`macos-package` is conditional on `Gate`, but GitHub treats a skipped job as
satisfying a required check — so gating it off (e.g. docs-only PRs) does not
block merges, while a real packaging failure does.

Note: these checks are currently NOT configured in the live repository settings — the `main` branch rules have no required status checks. They must be added.

## Additional Protection

- Restrict who can push to matching branches: enable for admins/automation policy.
- Require branches to be up to date before merging: recommended.
- Do not allow bypassing the above settings, unless explicitly needed for emergency admin flow.

## Release Flow

- Merge PRs to `main` only after `Test` and `Build` pass and one non-author approval is present.
- Create and push a semver tag from `main` to start a release:
  - Example: `git tag v1.2.3 && git push origin v1.2.3`
- Tag push triggers `.github/workflows/cli-release.yml`, which:
  - validates the tag (SemVer shape, and that the tagged commit is reachable from `main`),
  - runs the CI build via `.github/workflows/cli-ci.yml`,
  - creates a draft release and signs every artifact into it — per-platform binaries, four per-platform checksums files (`linux-checksums.txt`, `darwin-checksums.txt`, `windows-checksums.txt`, `bicep-checksums.txt`), and a `.sigstore.json` bundle alongside each asset,
  - publishes the release only after the `release` environment approval and the publish gate pass.

## Tag Protection

- Configure a `v*` tag ruleset restricting who can create tags: who can tag is who can ship.
- Enable immutable releases once the draft-then-publish flow is live.
