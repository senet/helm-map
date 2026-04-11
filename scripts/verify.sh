#!/usr/bin/env bash
# verify.sh — standalone checksum verification helper for helm-map releases.
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "Usage: $0 <file> <checksums.txt>" >&2
  exit 1
fi

FILE="$1"
CHECKSUMS="$2"
FILENAME="$(basename "$FILE")"

EXPECTED="$(grep "$FILENAME" "$CHECKSUMS" | awk '{print $1}')"
if [ -z "$EXPECTED" ]; then
  echo "No checksum found for $FILENAME in $CHECKSUMS" >&2
  exit 1
fi

ACTUAL="$(sha256sum "$FILE" | awk '{print $1}')"
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "FAIL: checksum mismatch for $FILENAME" >&2
  echo "  Expected: $EXPECTED" >&2
  echo "  Actual:   $ACTUAL" >&2
  exit 1
fi

echo "OK: $FILENAME checksum verified."
