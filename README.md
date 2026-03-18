# HiddenLayer Azure APIM Guardrails

A command-line tool for deploying and managing HiddenLayer AI security scanning in Azure API Management.

## Prerequisites

- **Azure CLI** - Must be installed and authenticated (`az login`)
- **HiddenLayer account** - Client ID, Secret, and Project ID from [console.hiddenlayer.ai](https://console.hiddenlayer.ai)

## Installation

### From Source

```bash
# Clone and build
git clone https://github.com/hiddenlayer/hiddenlayer-azure-apim-guardrails.git
cd hiddenlayer-azure-apim-guardrails/cli
make build

# Or install directly
go install github.com/hiddenlayer/hiddenlayer-azure-apim-guardrails/cli@latest
```

### Pre-built Binaries

Download from the [releases page](https://github.com/hiddenlayer/hiddenlayer-azure-apim-guardrails/releases).

## Quick Start

```bash
# 1. Authenticate with Azure
az login

# 2. Initialize configuration
hiddenlayer-apim init

# 3. Edit .env with your Azure and HiddenLayer credentials
vim .env

# 4. Deploy policy fragments to APIM
hiddenlayer-apim deploy

# 5. List available APIs
hiddenlayer-apim list

# 6. Apply HiddenLayer policy to an API
hiddenlayer-apim apply my-openai-api
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Create a `.env` configuration template |
| `deploy` | Deploy HiddenLayer fragments to APIM |
| `list` | List all APIs in the APIM instance |
| `apply <api-id>` | Apply HiddenLayer policy to an API |
| `remove <api-id>` | Remove HiddenLayer policy from an API |
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

# Optional
HL_TARGET_API=default-api-id
```

## Fragment Packages

Fragments are organized into **packages** under `cli/internal/policy/packages/`. Each package contains a `package.json` manifest (including `name`, `version`, `inbound`, `outbound`) and `.xml` fragment files. The CLI auto-discovers available packages at compile time via `//go:embed`.

- Use `--package <name>` on `deploy`, `apply`, `status`, or `remove` to select a specific package.
- Set `HL_PACKAGE` in your `.env` to set a default.
- If only one package is available, it is auto-selected.
- If multiple packages exist and no flag/env is set, an interactive menu is shown.

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
  --parameters ./hl-bicep/main.bicepparam
```

## How It Works

This CLI deploys policy fragments that use the HiddenLayer v1 Interactions API:

- **Single endpoint**: `POST /detection/v1/interactions` (called twice: once for input, once for output)

### Structured Body Format

The APIM policy constructs a structured JSON body for each call:

```json
{
  "metadata": { "model": "...", "requester_id": "...", "provider": "..." },
  "input": { "messages": [...] }
  "output": { "messages": [...] }
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
| `HL-Runtime-Edge-Provider` | Edge provider name used by `v2`/`v3` eval packages (defaults to `azure-apim`) |
| `HL-Requester-Id` | Requester identifier (defaults to subscription key or IP) |
| `Hl-Runtime-Session-Id` | Optional session/conversation identifier consumed by the policy and not forwarded to the backend |

The `HL-Runtime-Action` response header is set to `BLOCK` or empty for downstream clients.

### v2 Evaluation Headers

The v2 evaluation fragments send additional context headers to the HiddenLayer API:

| Header | Description |
|--------|-------------|
| `HL-Runtime-Edge-Provider` | Edge provider name forwarded to HiddenLayer; uses the client value when present, otherwise defaults to `azure-apim` |
| `HL-Runtime-Edge-Provider-Version` | Edge provider version (`0.1`) |
| `HL-Runtime-Edge-Provider-Metadata` | JSON object with APIM deployment context (API name, version, service name, region, API ID, revision, subscription name, operation ID) |
| `Hl-Runtime-Session-Id` | Session identifier for conversation tracking; forwards the client-provided `Hl-Runtime-Session-Id` value when present, otherwise sends an empty value |

## Examples

### Deploy and Apply

```bash
# Deploy fragments
hiddenlayer-apim deploy

# Apply to specific API
hiddenlayer-apim apply openai-proxy

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
# Remove HiddenLayer from an API
hiddenlayer-apim remove openai-proxy
```

## Building

```bash
cd cli

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
- `.github/workflows/cli-release.yml` runs when a semver tag is pushed (for example `v1.2.3`) and:
  - validates tag format,
  - runs tests,
  - builds cross-platform binaries + `checksums.txt`,
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

Branch protection setup for these requirements is documented in `.github/BRANCH_PROTECTION.md`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

## License

Copyright HiddenLayer, Inc. Licensed under the [Apache License 2.0](LICENSE).
