# HiddenLayer APIM CLI

A command-line tool for deploying and managing HiddenLayer AI security scanning in Azure API Management.

## Prerequisites

- **Azure CLI** - Must be installed and authenticated (`az login`)
- **HiddenLayer account** - Client ID, Secret, and Project ID from [console.hiddenlayer.ai](https://console.hiddenlayer.ai)

## Installation

### From Source

```bash
# Clone and build
git clone https://github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails.git
cd hiddenlayer-azure-apim-guardrails/cli
make build

# Or install directly
go install github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli@latest
```

### Pre-built Binaries

Download from the [releases page](https://github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/releases).

Release assets are published as versioned archives:

- macOS: `hiddenlayer-apim-vX.Y.Z-darwin-arm64.pkg`, `hiddenlayer-apim-vX.Y.Z-darwin-amd64.pkg`
- Linux: `hiddenlayer-apim-vX.Y.Z-linux-amd64.tar.gz`, `hiddenlayer-apim-vX.Y.Z-linux-arm64.tar.gz`
- Windows installer: `hiddenlayer-apim-vX.Y.Z-windows-amd64.msi`
- Windows portable archive: `hiddenlayer-apim-vX.Y.Z-windows-amd64.zip`
- Bicep bundle: `hiddenlayer-apim-vX.Y.Z-bicep.zip`

Each release also includes:

- `checksums.txt`
- `checksums.txt.sig` and `checksums.txt.pem`
- per-archive `.sig` and `.pem` files for cosign verification

macOS packages are Developer ID signed, notarized, and stapled. The `.pkg` installers place the binary in `/usr/local/bin`. Windows users should prefer the `.msi` installer, which installs the binary under `%ProgramFiles%\HiddenLayer\APIM CLI\` and adds that directory to the machine `PATH`. The Windows `.zip` remains available for portable use. Windows binaries and MSI packages are Authenticode signed with Azure Artifact Signing when the Azure signing configuration is present. All published archives, packages, and the checksum file are also signed with cosign keyless signing.

### Quick Start (Pre-built Binary)

```bash
# 1. Download the installer for your platform
# macOS example (Apple Silicon):
curl -L -O https://github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/releases/download/v1.2.3/hiddenlayer-apim-v1.2.3-darwin-arm64.pkg

# Windows example:
# curl.exe -L -O https://github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/releases/download/v1.2.3/hiddenlayer-apim-v1.2.3-windows-amd64.msi

# 2. Install it
# macOS installs to /usr/local/bin
sudo installer -pkg hiddenlayer-apim-v1.2.3-darwin-arm64.pkg -target /
# Windows installs to %ProgramFiles%\HiddenLayer\APIM CLI and updates machine PATH
# msiexec /i hiddenlayer-apim-v1.2.3-windows-amd64.msi

# 3. Verify the binary works
hiddenlayer-apim version

# 4. Initialize configuration
hiddenlayer-apim init

# 5. Edit .env with your Azure and HiddenLayer credentials
vim .env

# 6. Optional: list available fragment packages
hiddenlayer-apim packages

# 7. Deploy policy fragments to APIM
hiddenlayer-apim deploy
```

## Quick Start

```bash
# 1. Authenticate with Azure
az login

# 2. Initialize configuration
hiddenlayer-apim init

# 3. Edit .env with your Azure and HiddenLayer credentials
vim .env

# 4. Optional: list available fragment packages
hiddenlayer-apim packages

# 5. Deploy policy fragments to APIM
hiddenlayer-apim deploy

# 6. List available APIs
hiddenlayer-apim list

# 7. Apply HiddenLayer policy to an API
hiddenlayer-apim apply my-openai-api
```

### Quick Start With Release Bicep Templates

Some customers may prefer to deploy the pre-built Bicep bundles published in GitHub releases instead of running the CLI locally.

1. Download and extract the Bicep bundle from the desired GitHub release asset.
2. Choose the package directory to deploy, such as `v1-interactions/`, `v2-request-evals/`, or `v2-response-evals/`.
3. Set the only required Bicep parameter, `apimServiceName`, to your existing APIM instance name. You can either edit that package's `main.bicepparam` or pass the value with `--parameters`.
4. Use Azure CLI to preview and deploy one package at a time:

```bash
# Authenticate and select the target subscription
az login
az account set --subscription "<subscription-id>"

# Preview the deployment
az deployment group what-if \
  --resource-group <rg> \
  --template-file ./v2-request-evals/main.bicep \
  --parameters apimServiceName=<apim-name>

# Deploy the fragments
az deployment group create \
  --resource-group <rg> \
  --template-file ./v2-request-evals/main.bicep \
  --parameters apimServiceName=<apim-name>
```

The Bicep templates do not read `.env` and do not deploy HiddenLayer credentials. Before enabling the fragments on an API, create the required APIM named values using your approved secret-management process:

- `hl-client-id`
- `hl-client-secret` (mark as secret, or reference Key Vault)
- `hl-project-id`
- `hl-host`
- `hl-tenant-id`
- `hl-oauth-cache-seconds`

After deployment completes, the fragments are available in API Management. To enable them without running the CLI, open the target API in the Azure portal policy editor and add the package's `<include-fragment fragment-id="..." />` entries to the appropriate inbound and outbound policy sections. The package manifest and `fragments/` directory show which fragment IDs belong to each package.

Why choose one workflow over the other: Bicep is useful when your organization requires reviewable infrastructure templates and avoids local CLI access to secrets. The CLI is safer for interactive operations because it validates APIM state first, compares existing resources before overwriting, preserves unrelated API policy rules, and prompts before policy changes.

## Commands

| Command | Description |
|---------|-------------|
| `init` | Create a `.env` configuration template |
| `packages` | List available fragment packages |
| `deploy` | Deploy HiddenLayer fragments to APIM without overwriting existing resources unless `--overwrite` is used |
| `list` | List all APIs in the APIM instance |
| `apply <api-id>` | Apply HiddenLayer policy to an API, prompting before policy updates unless `--yes` is used |
| `remove <api-id>` | Remove selected HiddenLayer package fragments from an API; use `--all` to remove every detected HiddenLayer fragment |
| `status` | Check deployment status and verify configuration |
| `export bicep` | Export package fragments as Bicep + XML artifacts |
| `version` | Print version information |

## Configuration

The CLI reads configuration from environment variables or a `.env` file:

```bash
# Azure Configuration
RG=your-resource-group
APIM_NAME=your-apim-instance

# HiddenLayer Credentials
HL_CLIENT=your-client-id
HL_SECRET=your-client-secret
HL_PROJECT_ID=your-project-id
HL_TENANT_ID=your-tenant-id

# Optional
AZURE_SUBSCRIPTION_ID=your-subscription-id
HL_HOST=hiddenlayer.ai
# HL_OAUTH_CACHE_SECONDS=5
HL_TARGET_API=default-api-id
HL_PACKAGE=v2-request-evals
```

## Fragment Packages

Fragments are organized into **packages** under `internal/policy/packages/`. Each package contains a `package.json` manifest (including `name`, `version`, `inbound`, `outbound`) and `.xml` fragment files. The CLI auto-discovers available packages at compile time via `//go:embed`.

- Use `--package <name>` on `deploy`, `apply`, `status`, or `remove` to select a specific package.
- Use `hiddenlayer-apim packages` to list package names and their fragments.
- Use `deploy --packages <a,b,c>` to deploy multiple packages in a single command.
- Use `apply --packages <a,b,c>` to apply multiple packages in a single command.
- Use `remove --packages <a,b,c>` to remove multiple packages in a single command.
- Set `HL_PACKAGE` in your `.env` to set a default.
- If only one package is available, it is auto-selected.
- If multiple packages exist and no flag/env is set, an interactive menu is shown.

## Non-Destructive Deployment

`hiddenlayer-apim deploy` creates missing HiddenLayer APIM named values and policy fragments. If a named value or fragment already exists:

- matching resources are left unchanged
- existing secret named values are skipped because Azure does not return secret values for comparison
- differing non-secret named values or policy fragments cause the command to stop with guidance

Use `hiddenlayer-apim deploy --overwrite` when you intentionally want the CLI to update existing HiddenLayer-managed named values or replace existing HiddenLayer policy fragment XML. `apply` and `remove` operate on API policies by adding or removing HiddenLayer `<include-fragment>` references while preserving unrelated APIM policy rules.

## Exporting Bicep Artifacts

Use `export bicep` to generate an editable IaC bundle from a package:

```bash
# Export all fragments from a package
hiddenlayer-apim export bicep --package v1-interactions --out ./hl-bicep

# Export only inbound fragments
hiddenlayer-apim export bicep --package v1-interactions --group inbound --out ./hl-bicep-inbound
```

Generated output includes:

- `main.bicep` - APIM policy fragment resources using `loadTextContent('fragments/<id>.xml')`
- `main.bicepparam` - starter parameter file
- `fragments/*.xml` - editable policy fragment files

Deploy with Azure CLI:

```bash
az deployment group create \
  --resource-group <rg> \
  --template-file ./hl-bicep/main.bicep \
  --parameters apimServiceName=<apim-name>
```

The same Azure CLI workflow applies to pre-built Bicep bundles downloaded from GitHub releases. Release bundles contain one directory per package; deploy the package directory you want, for example `./v1-interactions/main.bicep` or `./v2-response-evals/main.bicep`. The only Bicep parameter is `apimServiceName`; the resource group is supplied by `az deployment group create --resource-group`.

## How It Works

This CLI deploys policy fragments that use a HiddenLayer evaluation API, depending on the selected package.

### v1 Interactions (`v1-interactions`)

- **Single endpoint**: `POST /detection/v1/interactions` (called twice: once for input, once for output)

### Structured Body Format

The APIM policy constructs a structured JSON body for each call:

```json
{
  "metadata": { "model": "...", "requester_id": "...", "provider": "..." },
  "input": { "messages": [...] }   // for input evaluation
  "output": { "messages": [...] }  // for output evaluation
}
```

Input messages are extracted from the OpenAI request body. Output messages are extracted from OpenAI `choices[].message` in the response.

### Enforcement

The enforcement signal comes from the `evaluation.action` field in the response body:

- **Allow**: Pass original payload unchanged
- **Alert**: Pass original payload (action logged server-side)
- **Redact**: Merge `modified_data.input.messages` or `modified_data.output.messages` back into the original payload
- **Block**: Return an OpenAI-format block response with threat information

### Metadata Headers

Optional override headers consumed by the input fragment and removed before calling the backend:

| Header | Description |
|--------|-------------|
| `HL-Model` | Override model identifier (otherwise read from request body) |
| `HL-Provider` | Provider name used by `v1-interactions` (defaults to `azure-apim`) |
| `HL-Runtime-Edge-Provider` | Edge provider name used by the bundled `v2` eval packages (defaults to `azure-apim`) |
| `HL-Requester-Id` | Requester identifier forwarded to HiddenLayer and removed before the backend call |
| `HL-Provider-Id` | Provider override forwarded to HiddenLayer and removed before the backend call by the v2 packages |
| `HL-Runtime-Session-Id` | Optional session/conversation identifier forwarded to HiddenLayer and removed before the backend call |

The `HL-Runtime-Action` response header is set to `BLOCK` or empty for downstream clients.

### v2 Evaluations (pass-through)

Two packages support pass-through request/response evaluation where HiddenLayer returns a provider-shaped payload:

- `v2-request-evals`
  - Calls `POST /detection/v2/request-evaluations` in **inbound**
  - Uses the `hl-runtime-action` response header from HiddenLayer to decide whether to block the backend call
- `v2-response-evals`
  - Calls `POST /detection/v2/response-evaluations` in **outbound**

In both packages, the APIM policy sends `HL-Runtime-Edge-Provider` to HiddenLayer, using the client value when present and otherwise defaulting to `azure-apim`. The policy surfaces the decision to clients as `HL-Runtime-Action: BLOCK` (or empty).

#### Headers Sent to HiddenLayer (v2)

The v2 evaluation fragments send additional context headers to the HiddenLayer API:

| Header | Description |
|--------|-------------|
| `HL-Roundtrip-Id` | Per-request identifier used to link request and response evaluations; generated by APIM when absent |
| `HL-Runtime-Edge-Provider` | Edge provider name forwarded to HiddenLayer; uses the client value when present, otherwise defaults to `azure-apim` |
| `HL-Runtime-Edge-Provider-Version` | Edge provider version (`0.1`) |
| `HL-Runtime-Edge-Provider-Metadata` | JSON object with APIM deployment context (API name, version, service name, region, API ID, revision, subscription name, operation ID) |
| `HL-Runtime-Session-Id` | Session identifier for conversation tracking; forwards the client-provided value when present, otherwise sends an empty value |
| `HL-Requester-Id` | Optional requester override consumed by Runtime Ingestion API |
| `HL-Provider-Id` | Optional provider override consumed by Runtime Ingestion API |

## Examples

### Deploy and Apply

```bash
# Deploy fragments
hiddenlayer-apim deploy

# Overwrite existing HiddenLayer APIM named values/fragments intentionally
hiddenlayer-apim deploy --overwrite

# Apply to specific API
hiddenlayer-apim apply openai-proxy

# Apply without an interactive confirmation prompt
hiddenlayer-apim apply openai-proxy --yes

# Apply with preview
hiddenlayer-apim apply openai-proxy --dry-run
```

### Check Status

```bash
# Verify everything is configured correctly
hiddenlayer-apim status
```

### Remove Policy

```bash
# Remove a selected HiddenLayer package from an API
hiddenlayer-apim remove openai-proxy

# Remove every detected HiddenLayer fragment from an API
hiddenlayer-apim remove openai-proxy --all
```

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install locally
make install
```

## CI/CD and Releases

GitHub Actions workflows are configured at the repository root:

- `.github/workflows/cli-ci.yml` runs on PRs to `main` and pushes to `main` (for `cli/**` changes), and executes:
  - `go test ./...`
  - `make build`
  - Windows MSI build and install/uninstall smoke tests
- `.github/workflows/cli-release.yml` runs when a semver tag is pushed (for example `v1.2.3`) and:
  - validates tag format,
  - runs tests,
  - builds Linux release archives, Windows `.msi` installers and `.zip` portable archives, and macOS `.pkg` installers,
  - Developer ID signs, notarizes, and staples macOS packages,
  - Authenticode-signs the Windows binary and MSI with Azure Artifact Signing when Azure signing is configured,
  - smoke-tests the Windows MSI on a GitHub-hosted Windows runner,
  - generates `checksums.txt` and cosign-signs each archive/package plus the checksum file,
  - publishes GitHub Release assets.

### Recommended Merge and Release Flow

1. Push feature branch.
2. Open PR to `main`.
3. Wait for required checks (`Test`, `Build`) to pass.
4. Get at least one non-author approval.
5. Merge to `main`.
6. When ready to release, create a tag from `main` and push it:

```bash
git checkout main
git pull
git tag v1.2.3
git push origin v1.2.3
```

The release workflow can also be manually dispatched with an RC version such as `v1.2.3-rc.1` to validate a branch build through the GitHub Release path before final release.

Branch protection setup for these requirements is documented in `.github/BRANCH_PROTECTION.md`.

## License

Copyright HiddenLayer, Inc.
