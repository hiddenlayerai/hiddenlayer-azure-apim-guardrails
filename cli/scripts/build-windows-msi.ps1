param(
    [string]$Version = $env:GITHUB_REF_NAME,
    [string]$GitCommit = "",
    [string]$BuildDate = "",
    [switch]$SkipBinaryBuild
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = "v0.1.0-dev"
}

if ([string]::IsNullOrWhiteSpace($GitCommit)) {
    try {
        $GitCommit = (git rev-parse --short HEAD).Trim()
    } catch {
        $GitCommit = "unknown"
    }
}

if ([string]::IsNullOrWhiteSpace($BuildDate)) {
    $BuildDate = [DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")
}

if ($Version -notmatch '^v?(?<major>[0-9]+)\.(?<minor>[0-9]+)\.(?<patch>[0-9]+)(?:[-+].*)?$') {
    throw "Version '$Version' is not compatible with Windows Installer ProductVersion. Expected vMAJOR.MINOR.PATCH[-PRERELEASE]."
}

$productVersion = "$($Matches.major).$($Matches.minor).$($Matches.patch)"
$dist = Join-Path $PWD "dist"
$binaryDir = Join-Path $dist "windows-amd64"
$binary = Join-Path $binaryDir "hiddenlayer-apim.exe"
$outputName = "hiddenlayer-apim-windows-amd64-$Version"
$msi = Join-Path $dist "$outputName.msi"

if (!$SkipBinaryBuild) {
    Remove-Item -Recurse -Force $binaryDir -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force -Path $binaryDir | Out-Null

    $ldflags = "-X github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/cmd.Version=$Version -X github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/cmd.GitCommit=$GitCommit -X github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/cmd.BuildDate=$BuildDate"

    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    try {
        go build -ldflags $ldflags -o $binary .
    }
    finally {
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    }
}

if (!(Test-Path $binary)) {
    throw "Windows binary not found: $binary"
}

dotnet build packaging\windows\HiddenLayer.Apim.Installer.wixproj `
    -c Release `
    -p:ProductVersion=$productVersion `
    -p:GitTag=$Version `
    -p:SourceBinary=$binary `
    -p:OutputPath="$dist\" `
    -p:OutputName=$outputName

if (!(Test-Path $msi)) {
    throw "Expected MSI was not produced: $msi"
}

Write-Host "Built MSI: $msi"
