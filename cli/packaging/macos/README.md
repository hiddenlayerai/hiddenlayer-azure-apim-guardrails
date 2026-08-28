# macOS packaging for `hiddenlayer-apim`

Builds the `hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg` installer. It is a
**product archive** wrapping a single component pkg (identifier
`ai.hiddenlayer.apim-cli`) that installs the `hiddenlayer-apim` universal
binary into `/usr/local/bin`.

The component is rooted at its install-location (rather than a payload rooted
at `/`), so its BOM never records the shared `/usr/local` parent — installing
can never re-stamp `/usr/local`'s mode/owner.

## Building

Local, ad-hoc signed (good for dev machines / VMs; not notarizable). Run from
the repo root:

```sh
cli/packaging/macos/build-pkg.sh \
  --version v0.1.0 \
  --binary build/hiddenlayer-apim-universal \
  --out dist
```

Fully signed (Developer ID certs in an unlocked keychain):

```sh
cli/packaging/macos/build-pkg.sh \
  --version v0.1.0 \
  --binary build/hiddenlayer-apim-universal \
  --identity "Developer ID Application: HiddenLayer, Inc. (TEAMID)" \
  --installer-identity "Developer ID Installer: HiddenLayer, Inc. (TEAMID)" \
  --out dist
```

The binary is codesigned with the hardened runtime (`--timestamp
--options runtime`); the component pkg is built with `pkgbuild`, wrapped into a
product archive with `productbuild`, then signed with `productsign`. The
unsigned intermediate is deleted. Signing is optional in both places: with no
identities the script produces an ad-hoc-signed binary in an unsigned pkg (this
is what pre-merge CI builds). Notarization and stapling happen in the release
workflow, not in this script.

## Why `DeveloperIDG2CA.cer` is vendored here

The release workflow imports this Apple intermediate ("Developer ID
Certification Authority", OU=G2) into the signing keychain alongside our leaf
certificates. Without it the Developer ID chain doesn't validate in the
ephemeral CI keychain, and `security find-identity -v` silently comes up
empty — the leaf certs are present but unusable, and signing fails with a
misleading "no identity found".

It is Apple's public intermediate from
<https://www.apple.com/certificateauthority/DeveloperIDG2CA.cer>. When
refreshing it, verify against the recorded digest:

```
shasum -a 256 DeveloperIDG2CA.cer
f16cd3c54c7f83cea4bf1a3e6a0819c8aaa8e4a1528fd144715f350643d2df3a  DeveloperIDG2CA.cer
```

and sanity-check the subject:

```sh
openssl x509 -inform der -in DeveloperIDG2CA.cer -noout -subject
# subject=CN=Developer ID Certification Authority, OU=G2, O=Apple Inc., C=US
```

## Verifying a release pkg

Hard checks — these must pass on a signed, notarized, stapled release pkg:

```sh
pkgutil --check-signature hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg
xcrun stapler validate hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg
```

Advisory only — `spctl`'s assessment of installer packages shifts across macOS
majors, so treat its output as informational rather than pass/fail:

```sh
spctl -a -vvv -t install hiddenlayer-apim-vX.Y.Z-darwin-universal.pkg
```
