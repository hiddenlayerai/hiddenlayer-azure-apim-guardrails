#!/bin/bash
set -euo pipefail

# Build (and optionally sign) the macOS installer pkg for the hiddenlayer-apim CLI.
#
# Usage (run from the repo root):
#   cli/packaging/macos/build-pkg.sh --version vX.Y.Z --binary /path/to/hiddenlayer-apim-universal \
#     [--identity "Developer ID Application: ..."] \
#     [--installer-identity "Developer ID Installer: ..."] \
#     [--out DIR]
#
# Without --identity the binary is ad-hoc signed (codesign -s -). Without
# --installer-identity the product archive is left unsigned (still installable
# locally / notarization not possible). Pre-merge CI reuses this script with no
# identities, so missing identities are never an error.

# The pkg ships a single location (the binary on PATH), so it is a product
# archive (productbuild) wrapping one component pkg rooted at its own
# install-location. The component's BOM is rooted at /usr/local/bin, so it
# never records the shared /usr/local parent — installing can never re-stamp
# /usr/local's mode/owner (the Homebrew footgun a payload rooted at / would hit).
#
# The product archive uses the raw synthesized Distribution.xml deliberately —
# do not hand-edit/"improve" it. The binary is universal, so a
# hostArchitectures constraint is unnecessary.
IDENTIFIER="ai.hiddenlayer.apim-cli"
INSTALL_LOCATION="/usr/local/bin"
BINARY_NAME="hiddenlayer-apim"

usage() {
  cat >&2 <<EOF
Usage: $(basename "$0") --version vX.Y.Z --binary /path/to/hiddenlayer-apim-universal
         [--identity "Developer ID Application: ..."]
         [--installer-identity "Developer ID Installer: ..."]
         [--out DIR]            (default: dist/)
EOF
  exit 2
}

VERSION=""
BINARY=""
IDENTITY=""
INSTALLER_IDENTITY=""
OUT_DIR="dist"

while [ $# -gt 0 ]; do
  case "$1" in
    --version)            VERSION="${2:?missing value for --version}"; shift 2 ;;
    --binary)             BINARY="${2:?missing value for --binary}"; shift 2 ;;
    --identity)           IDENTITY="${2:?missing value for --identity}"; shift 2 ;;
    --installer-identity) INSTALLER_IDENTITY="${2:?missing value for --installer-identity}"; shift 2 ;;
    --out)                OUT_DIR="${2:?missing value for --out}"; shift 2 ;;
    -h|--help)            usage ;;
    *) echo "Unknown argument: $1" >&2; usage ;;
  esac
done

[ -n "$VERSION" ] || { echo "--version is required" >&2; usage; }
[ -n "$BINARY" ]  || { echo "--binary is required" >&2; usage; }

# Full SemVer prerelease grammar (numeric ids reject leading zeros);
# keep in sync with the validate job in .github/workflows/cli-release.yml.
SEMVER_RE='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?$'
if [[ ! "$VERSION" =~ $SEMVER_RE ]]; then
  echo "Version '$VERSION' is not a valid semver tag (expected vMAJOR.MINOR.PATCH[-PRERELEASE])." >&2
  exit 1
fi

if [ ! -f "$BINARY" ]; then
  echo "Binary not found: $BINARY" >&2
  exit 1
fi

# Apple's timestamp service intermittently flakes, and codesign/productsign
# hard-fail when it is unreachable — retry with backoff before giving up.
retry() {
  local max="$1"
  shift
  local attempt=1 delay=5
  while true; do
    if "$@"; then
      return 0
    fi
    if [ "$attempt" -ge "$max" ]; then
      return 1
    fi
    echo "Attempt $attempt/$max failed ($1); retrying in ${delay}s..." >&2
    sleep "$delay"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

BIN_ROOT="$(mktemp -d -t hl-apim-pkg-bin-root)"
COMPONENTS_DIR="$(mktemp -d -t hl-apim-pkg-components)"
cleanup() {
  rm -rf "$BIN_ROOT" "$COMPONENTS_DIR"
  # If signing was requested, UNSIGNED_PKG is an intermediate in OUT_DIR that the
  # productsign step removes on success. A failed productsign would otherwise
  # leave it behind as a stray unsigned artifact, so clean it here too. Never
  # touch it when it IS the final pkg (an unsigned build, where they're equal).
  if [ -n "${UNSIGNED_PKG:-}" ] && [ "${UNSIGNED_PKG:-}" != "${FINAL_PKG:-}" ]; then
    rm -f "$UNSIGNED_PKG"
  fi
}
trap cleanup EXIT

# Stage the binary payload (binary as hiddenlayer-apim, mode 0755).
# Copy without extended attributes (-X) so stray xattrs (e.g. quarantine)
# don't end up encoded as AppleDouble (._*) entries in the pkg archive.
# Note: the SIP-managed com.apple.provenance xattr cannot be avoided on
# provenance-tagged dev shells; it is harmless — the installer reapplies it
# as an xattr on the installed file, never as a literal ._ file.
cp -X "$BINARY" "$BIN_ROOT/$BINARY_NAME"
chmod 0755 "$BIN_ROOT/$BINARY_NAME"

# Make the install-location directory world-traversable. pkgbuild stamps the
# payload-root dir's own mode onto the install-location (/usr/local/bin), and
# mktemp -d created BIN_ROOT as 0700. This only bites when THIS pkg is the
# first thing to create /usr/local/bin on the machine: pkgbuild then creates it
# drwx------ root:wheel and non-root users can't traverse it to exec the
# binary. (If /usr/local/bin already exists, the installer leaves its mode
# untouched.) Force 0755 so a first-creator install matches a normal
# /usr/local/bin.
chmod 0755 "$BIN_ROOT"

sign_binary() {
  codesign -s "$IDENTITY" \
    --timestamp \
    --options runtime \
    -f "$BIN_ROOT/$BINARY_NAME"
}

# Sign the binary (Developer ID with hardened runtime, or ad-hoc fallback).
if [ -n "$IDENTITY" ]; then
  echo "Signing binary with Developer ID identity: $IDENTITY"
  if ! retry 3 sign_binary; then
    echo "codesign failed after 3 attempts (Apple timestamp service unreachable?)." >&2
    exit 1
  fi
  # codesign reports a verified secure timestamp as a "Timestamp=" line; a
  # "Signed Time" line instead means the signature carries only an unverified
  # local time (no secure timestamp) and notarization would reject it.
  SIGN_INFO="$(codesign -dvv "$BIN_ROOT/$BINARY_NAME" 2>&1)"
  if ! printf '%s\n' "$SIGN_INFO" | grep -q '^Timestamp='; then
    echo "Binary signature lacks a secure timestamp (no Timestamp= in codesign -dvv):" >&2
    printf '%s\n' "$SIGN_INFO" >&2
    exit 1
  fi
else
  echo "No --identity given; ad-hoc signing the binary."
  codesign -s - -f "$BIN_ROOT/$BINARY_NAME"
fi

mkdir -p "$OUT_DIR"

PKG_NAME="${BINARY_NAME}-${VERSION}-darwin-universal.pkg"
FINAL_PKG="$OUT_DIR/$PKG_NAME"

if [ -n "$INSTALLER_IDENTITY" ]; then
  UNSIGNED_PKG="$OUT_DIR/${BINARY_NAME}-${VERSION}-darwin-universal-unsigned.pkg"
else
  UNSIGNED_PKG="$FINAL_PKG"
fi

# productbuild/productsign refuse to overwrite an existing destination; clear
# leftovers so a same-version rebuild into an existing --out succeeds.
rm -f "$FINAL_PKG" "$OUT_DIR/${BINARY_NAME}-${VERSION}-darwin-universal-unsigned.pkg"

COMPONENT_PKG="$COMPONENTS_DIR/${BINARY_NAME}.pkg"

pkgbuild \
  --root "$BIN_ROOT" \
  --identifier "$IDENTIFIER" \
  --version "${VERSION#v}" \
  --install-location "$INSTALL_LOCATION" \
  "$COMPONENT_PKG"

# Wrap the component in a product archive. Synthesizing the distribution keeps
# the pkg-ref/version in sync with the component automatically.
DIST_XML="$COMPONENTS_DIR/distribution.xml"
productbuild --synthesize \
  --package "$COMPONENT_PKG" \
  "$DIST_XML"

productbuild \
  --distribution "$DIST_XML" \
  --package-path "$COMPONENTS_DIR" \
  "$UNSIGNED_PKG"

sign_product() {
  # productsign refuses to overwrite an existing destination, and a failed
  # attempt can leave a partial file behind — remove it before every attempt
  # so a retry never trips over a stale partial.
  rm -f "$FINAL_PKG"
  productsign \
    --sign "$INSTALLER_IDENTITY" \
    --timestamp \
    "$UNSIGNED_PKG" \
    "$FINAL_PKG"
}

if [ -n "$INSTALLER_IDENTITY" ]; then
  echo "Signing pkg with installer identity: $INSTALLER_IDENTITY"
  if ! retry 3 sign_product; then
    # A partial destination must never survive as a plausible artifact.
    rm -f "$FINAL_PKG"
    echo "productsign failed after 3 attempts (Apple timestamp service unreachable?)." >&2
    exit 1
  fi
  rm "$UNSIGNED_PKG"
else
  echo "No --installer-identity given; pkg is unsigned."
fi

echo "Built: $FINAL_PKG"
