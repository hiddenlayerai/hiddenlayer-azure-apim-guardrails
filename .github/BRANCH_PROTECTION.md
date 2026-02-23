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

These checks are produced by `.github/workflows/cli-ci.yml`.

## Additional Protection

- Restrict who can push to matching branches: enable for admins/automation policy.
- Require branches to be up to date before merging: recommended.
- Do not allow bypassing the above settings, unless explicitly needed for emergency admin flow.

## Release Flow

- Merge PRs to `main` only after `Test` and `Build` pass and one non-author approval is present.
- Create and push a semver tag from `main` to publish release binaries:
  - Example: `git tag v1.2.3 && git push origin v1.2.3`
- Tag push triggers `.github/workflows/cli-release.yml`, which:
  - validates tag format,
  - runs tests,
  - builds release artifacts under `cli/dist/`,
  - publishes a GitHub Release with attached binaries and checksums.
