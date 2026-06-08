param(
    [Parameter(Mandatory = $true)]
    [string]$MsiPath,

    [string]$InstallLog = "$env:TEMP\hiddenlayer-apim-install.log",
    [string]$UninstallLog = "$env:TEMP\hiddenlayer-apim-uninstall.log"
)

$ErrorActionPreference = "Stop"

$resolvedMsi = (Resolve-Path $MsiPath).Path
$installDir = Join-Path $env:ProgramFiles "HiddenLayer\APIM CLI"
$binary = Join-Path $installDir "hiddenlayer-apim.exe"
$installSucceeded = $false

function Normalize-PathEntry {
    param([string]$PathEntry)

    if ([string]::IsNullOrWhiteSpace($PathEntry)) {
        return ""
    }

    return $PathEntry.Trim().TrimEnd("\", "/")
}

function Test-PathEntry {
    param(
        [string]$PathValue,
        [string]$ExpectedEntry
    )

    $normalizedExpected = Normalize-PathEntry $ExpectedEntry
    foreach ($entry in ($PathValue -split ";")) {
        if ((Normalize-PathEntry $entry) -ieq $normalizedExpected) {
            return $true
        }
    }

    return $false
}

function Invoke-Msiexec {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments,

        [Parameter(Mandatory = $true)]
        [string]$FailureMessage
    )

    $process = Start-Process -FilePath "msiexec.exe" -ArgumentList $Arguments -Wait -PassThru
    if ($process.ExitCode -notin @(0, 3010)) {
        throw "$FailureMessage Exit code: $($process.ExitCode)"
    }
}

try {
    Invoke-Msiexec `
        -Arguments @("/i", "`"$resolvedMsi`"", "/qn", "/norestart", "/l*v", "`"$InstallLog`"") `
        -FailureMessage "MSI install failed."
    $installSucceeded = $true

    if (!(Test-Path $binary)) {
        throw "Expected installed binary was not found: $binary"
    }

    & $binary version
    if ($LASTEXITCODE -ne 0) {
        throw "Installed binary failed version check. Exit code: $LASTEXITCODE"
    }

    $machinePath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
    if (!(Test-PathEntry -PathValue $machinePath -ExpectedEntry $installDir)) {
        throw "Machine PATH does not contain install directory: $installDir"
    }

    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    $env:PATH = "$machinePath;$userPath"

    $pathCheck = powershell -NoProfile -Command "Get-Command hiddenlayer-apim -ErrorAction Stop | Select-Object -ExpandProperty Source"
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($pathCheck)) {
        throw "hiddenlayer-apim was not available on PATH from a new shell."
    }

    Write-Host "Installed binary resolved from PATH: $pathCheck"
}
finally {
    if ($installSucceeded) {
        Invoke-Msiexec `
            -Arguments @("/x", "`"$resolvedMsi`"", "/qn", "/norestart", "/l*v", "`"$UninstallLog`"") `
            -FailureMessage "MSI uninstall failed."

        if (Test-Path $binary) {
            throw "Installed binary still exists after uninstall: $binary"
        }

        $machinePath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
        if (Test-PathEntry -PathValue $machinePath -ExpectedEntry $installDir) {
            throw "Machine PATH still contains install directory after uninstall: $installDir"
        }

        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        $env:PATH = "$machinePath;$userPath"
        $postUninstallPathCheck = powershell -NoProfile -Command "Get-Command hiddenlayer-apim -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source"
        if (![string]::IsNullOrWhiteSpace($postUninstallPathCheck)) {
            throw "hiddenlayer-apim was still available on PATH after uninstall: $postUninstallPathCheck"
        }
    }
}
