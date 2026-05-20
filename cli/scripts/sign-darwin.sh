#!/bin/bash
set -euo pipefail

# Sign a darwin Mach-O binary with Developer ID or fall back to ad-hoc signing.

BINARY="$1"

if [ -z "${APPLE_SIGNING_IDENTITY:-}" ]; then
  echo "APPLE_SIGNING_IDENTITY not set, using ad-hoc signing for $BINARY"
  codesign -s - -f "$BINARY"
else
  echo "Signing $BINARY with Developer ID identity: $APPLE_SIGNING_IDENTITY"
  codesign -s "$APPLE_SIGNING_IDENTITY" \
    --timestamp \
    --options runtime \
    -f "$BINARY"
fi
