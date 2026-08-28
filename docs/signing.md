# Signing `hiddenlayer-apim` release artifacts

How release artifacts are signed, notarized, and attested, what configuration
the release workflow requires, and how consumers verify what they download.
Signing claims apply to **release assets only** — `go install` (see the
README) is a legitimately unsigned path.

## Signing policy

PR and `main` builds are always **unsigned**; signed artifacts are produced
only by `cli-release.yml`.

| Context | Workflow | Output |
|---|---|---|
| PRs / merges to `main` | `cli-ci.yml` | unsigned binaries reporting a dev version, uploaded as artifacts |
| Packaging-path PRs, or any PR with the `build-packages` label | `packaging-ci.yml` | unsigned pkg + BOM verify + real `installer -pkg` smoke test, with zero secrets |
| Tag push `v*` | `cli-release.yml` | signed, notarized, attested assets in a draft release, then published |

Key properties of the release path:

- **We sign what we tested.** `cli-release.yml`'s build job calls
  `cli-ci.yml` — the same test-then-build pipeline that runs on every PR — and
  the platform jobs download those exact artifacts from their own run. Nothing
  is rebuilt outside CI and nothing is selected from a run we did not start.
- **Signing is a commitment.** There is no dispatch or dry-run mode that
  signs something we do not intend to ship. The unsigned pre-merge exercise of
  the packaging surface is `packaging-ci.yml`'s job.
- **A release never degrades to unsigned output.** Each platform job
  preflights its signing configuration and hard-fails naming the missing
  secret or variable; every later failure branch (identity discovery,
  notarization, post-sign verification) exits nonzero. Unsigned artifacts are
  already available from `packaging-ci.yml`.
- **Nothing becomes public until the end.** Assets are signed and uploaded
  into a *draft* release; a single `publish` job verifies the full inventory
  and flips it public.

## Cutting a release (maintainers)

Push a SemVer tag: `git tag v1.2.3 && git push origin v1.2.3`. The pipeline
does the rest: validate → build → create one draft → four platform signing
jobs → publish.

- **Tag from `main` only.** The `validate` job compares the tag against
  `main` and fails unless the tagged commit is on it. Hotfix tags off other
  branches are not supported.
- **Never tag a pre-rewrite commit.** A tag push executes the workflow file
  *at the tagged commit* — tagging any commit that predates this pipeline runs
  the old warn-and-publish workflow stored there, with none of these guards.
  If an old commit must ship, cherry-pick onto `main` and tag that.
- **One draft, created once.** `create_draft` creates (or, on a re-run,
  adopts) exactly one draft for the tag and fails on a published release or
  duplicate drafts. All later mutations address it by release ID, never by tag.
- **Human approval.** The `release` environment carries required reviewers.
  Reviewers may be asked to approve more than once as successive waves of jobs
  reach the environment (`create_draft` → the platform jobs → `publish`); the
  exact approval texture gets confirmed at the rc dry-run.
- **First-run guard: `RELEASE_PIPELINE_VALIDATED`.** `publish` refuses to
  flip the draft until this repository variable is `true`. A human sets it
  once, after verifying a release candidate's draft end-to-end with the
  recipe below. It stays set — from then on, **publishing is automatic on tag
  push** (explicit policy; the per-release human step is the environment
  approval).
- **Prerelease tags publish as prereleases.** A hyphenated suffix
  (`v1.2.3-rc1`) sets the release's prerelease flag; prereleases are never
  `latest`.

## Release assets

Ten assets per release, each with a `<asset>.sigstore.json` Sigstore bundle
alongside (20 files total; the four checksums files are signed too):

- `hiddenlayer-apim-vX.Y.Z-linux-amd64.tar.gz`, `…-linux-arm64.tar.gz`
- `hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg` — signed, notarized, stapled installer
- `hiddenlayer-apim-vX.Y.Z-darwin-universal.tar.gz` — sideload tarball of the
  **same signed universal binary** the pkg carries (the pkg's notarization
  ticket covers its cdhash; no separate signing)
- `hiddenlayer-apim-vX.Y.Z-windows-amd64.zip` — Authenticode-signed `hiddenlayer-apim.exe`
- `hiddenlayer-apim-vX.Y.Z-bicep.zip` — Bicep policy packages at the archive root
- `linux-checksums.txt`, `darwin-checksums.txt`, `windows-checksums.txt`, `bicep-checksums.txt`

macOS: the binary is Developer ID Application-signed (hardened runtime, secure
timestamp), packaged, `productsign`ed with the Developer ID Installer cert,
notarized via `notarytool`, and stapled. Windows: Azure Artifact Signing over
OIDC — no certificate secrets to store — with an RFC 3161 timestamp and an
in-workflow chain verification. Every asset is keyless-signed with cosign v3
using the workflow's GitHub OIDC identity; the bundle carries signature,
certificate, and Rekor inclusion proof in one file.

## Verifying a release

```sh
# Integrity. Each checksums file lists that platform's sibling assets, so
# --ignore-missing skips the ones you did not download.
sha256sum -c --ignore-missing linux-checksums.txt
shasum -a 256 -c --ignore-missing darwin-checksums.txt   # macOS

# Provenance (repeat per asset). The identity pins the exact repo, the
# release workflow, and a tag ref — not just the org.
cosign verify-blob \
  --bundle hiddenlayer-apim-vX.Y.Z-linux-amd64.tar.gz.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/\.github/workflows/cli-release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  hiddenlayer-apim-vX.Y.Z-linux-amd64.tar.gz
```

macOS pkg:

```sh
pkgutil --check-signature hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg
xcrun stapler validate hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg
# ADVISORY only: spctl can lag Apple's CDN/policy state; the two checks
# above are the gate.
spctl -a -vvv -t install hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg
```

Windows:

```powershell
Get-AuthenticodeSignature .\hiddenlayer-apim.exe   # Status must be Valid
```

### Quarantine and the darwin sideload tarball

Flat Mach-O binaries cannot be stapled — only bundles and packages carry
tickets. The sideload tarball works anyway because the pkg's notarization
ticket covers the binary's cdhash: on an online machine, Gatekeeper looks the
hash up with Apple and passes. On an **offline** machine, clear quarantine
after extracting:

```sh
xattr -d com.apple.quarantine ./hiddenlayer-apim
```

Whether that step is needed depends on transport: `curl`/`wget` downloads
never get the quarantine xattr; browser, Slack, and Archive Utility transfers
do, and macOS `bsdtar` propagates the xattr from a quarantined archive onto
extracted files.

## Version strings: where the `v` lives

| Surface | Form | Set by |
|---|---|---|
| git tag | `vX.Y.Z[-suffix]` | maintainer; `validate` enforces SemVer |
| release asset filenames | keep the `v` (`…-vX.Y.Z-…`) | release jobs, from the tag name |
| binary `version` output | `X.Y.Z` (stripped) | build stamps `${VERSION#v}` |
| `pkgbuild --version` | `X.Y.Z` (stripped) | pkg versions must start with a digit |
| dev / Makefile builds | `v0.1.0-dev+<commit>` (keep the `v`) | `cli/Makefile` fallback |

The asymmetry is deliberate: filenames and tags carry the `v`, the binary and
pkg metadata do not, and dev builds keep it so they are unmistakable. Do not
"fix" one side to match the other. Each platform job asserts the binary it is
about to sign reports exactly `${TAG#v}` before signing it.

## GitHub configuration

### Repository secrets (10)

| Secret | Used by | Purpose |
|---|---|---|
| `APPLE_CERTIFICATE_BASE64` / `APPLE_CERTIFICATE_PASSWORD` | macOS | Developer ID **Application** cert (.p12) and its export password — signs the binary |
| `APPLE_INSTALLER_CERTIFICATE_BASE64` / `APPLE_INSTALLER_CERTIFICATE_PASSWORD` | macOS | Developer ID **Installer** cert (.p12) and its password — signs the pkg |
| `APPLE_API_KEY_BASE64` / `APPLE_API_KEY_ID` / `APPLE_API_ISSUER_ID` | macOS | App Store Connect API key (.p8), key ID, and issuer UUID for `notarytool` |
| `AZURE_CLIENT_ID` / `AZURE_TENANT_ID` / `AZURE_SUBSCRIPTION_ID` | Windows | Entra app registration for the Artifact Signing OIDC login |

### Repository variables

| Variable | Purpose |
|---|---|
| `AZURE_CODESIGN_ENDPOINT` | Artifact Signing account endpoint URL (region-specific) |
| `AZURE_CODESIGN_ACCOUNT_NAME` | Artifact Signing account name |
| `AZURE_CODESIGN_PROFILE_NAME` | Certificate profile within the account |
| `RELEASE_PIPELINE_VALIDATED` | First-run publish guard (see above) |

The vars-vs-secrets split is deliberate: the three `AZURE_CODESIGN_*` values
only *identify* the signing account and are useless without an OIDC token, so
they are plain variables — visible in logs, which makes misconfiguration
debuggable in a public repo. Everything that authenticates is a secret.

Secrets are repo-level, not environment-scoped, so maintainers can view and
rotate them. The `release` environment still does the gating work: its
deployment policy restricts which refs may run the signing jobs, its required
reviewers add the human approval, and it forms the Azure federated-credential
subject (`…:environment:release`) — so the Windows signing credential is
unusable outside environment-gated jobs regardless of where the secret values
live. This repo is public; GitHub never exposes secrets or the `release`
environment to fork PRs.

## Re-running a failed release

- Use **"Re-run failed jobs"** only. Every mutation is ID-addressed and
  uploads are idempotent (a same-name asset from a previous attempt is
  deleted before re-upload), so re-running a platform job is safe.
- A macOS re-run **re-signs**, producing new bytes and therefore a fresh
  notary submission. That is correct behavior, not waste — checksums and
  bundles are regenerated from the new bytes in the same job. (BuildDate is
  stamped from the commit date, so the underlying binaries are stable across
  re-runs.)
- If notarization times out or fails, the submission id is in the failed
  job's summary. `xcrun notarytool wait <id>` / `notarytool log <id>` are
  **manual investigation tools** for deciding what went wrong; the CI retry
  path is still "Re-run failed jobs".
- Re-runs are valid **only while the release is a draft**. Every platform job
  and `publish` assert draft-ness and refuse to touch a published release.

## Remediation

There is no yank. Cosign bundles and their Rekor transparency-log entries are
permanent, and published assets should be assumed downloaded.

- **Bad draft** (never published): delete the release and the tag; the tag is
  burned — document it and move on to the next patch version.
- **Bad published release**: before immutable releases are enabled, assets or
  the release can be deleted; either way the durable remediation is to publish
  a fixed version and document the burned tag in its notes.
- **Duplicate drafts**: `create_draft` refuses to proceed if more than one
  release exists for the tag — delete the stray drafts and re-run.

## Design deviations from typical setups

- **Draft-then-publish instead of publish-triggered**: all assets attach and
  verify before anything is public, which is also what immutable releases
  require.
- **`gh api`/REST with ID-addressed mutations instead of a release-uploader
  action**: parallel draft creation via uploader actions provably races; a
  single creator plus ID-addressing removes the tag-lookup TOCTOU.
- **`<asset>.sigstore.json` bundles instead of detached `.sig`/`.pem`**:
  cosign v3's self-contained bundle format (signature + cert + Rekor proof).
- **`spctl` is advisory**: CI runners and fresh machines can lag Apple's
  policy CDN; `stapler validate` + `pkgutil --check-signature` are the gate.
