# Windows MSI Packaging

This directory contains the WiX project used to build the Windows MSI for `hiddenlayer-apim`.

The MSI installs `hiddenlayer-apim.exe` to:

```text
%ProgramFiles%\HiddenLayer\APIM CLI\
```

It also adds that directory to the machine `PATH`, so a new shell can run:

```powershell
hiddenlayer-apim version
```

## Build

Build the Windows binary first, then build the installer:

```powershell
$version = "v1.2.3-rc.1"
$productVersion = "1.2.3"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o dist\windows-amd64\hiddenlayer-apim.exe .
Remove-Item Env:GOOS
Remove-Item Env:GOARCH

dotnet build packaging\windows\HiddenLayer.Apim.Installer.wixproj `
  -c Release `
  -p:ProductVersion=$productVersion `
  -p:GitTag=$version `
  -p:SourceBinary="$PWD\dist\windows-amd64\hiddenlayer-apim.exe" `
  -p:OutputPath="$PWD\dist\" `
  -p:OutputName="hiddenlayer-apim-$version-windows-amd64"
```

The release workflow signs the `.exe` before MSI packaging when Azure signing is configured, then signs the `.msi` after packaging.
