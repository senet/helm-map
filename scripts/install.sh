#!/usr/bin/env bash
set -euo pipefail

PLUGIN_VERSION="0.1.0"
PROJECT_NAME="helm-map"
GITHUB_REPO="senet/helm-map"

# Detect OS.
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  darwin|linux) ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture.
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${PLUGIN_VERSION}/${PROJECT_NAME}_${OS}_${ARCH}.tar.gz"
CHECKSUM_URL="https://github.com/${GITHUB_REPO}/releases/download/v${PLUGIN_VERSION}/checksums.txt"

echo "Installing helm-map v${PLUGIN_VERSION} for ${OS}/${ARCH}..."

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Download the binary archive and checksums.
curl -sSL "$DOWNLOAD_URL" -o "${TMPDIR}/${PROJECT_NAME}.tar.gz"
curl -sSL "$CHECKSUM_URL" -o "${TMPDIR}/checksums.txt"

# Verify checksum.
EXPECTED_HASH="$(grep "${PROJECT_NAME}_${OS}_${ARCH}.tar.gz" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
if [ -n "$EXPECTED_HASH" ]; then
  ACTUAL_HASH="$(sha256sum "${TMPDIR}/${PROJECT_NAME}.tar.gz" | awk '{print $1}')"
  if [ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]; then
    echo "Checksum verification failed!" >&2
    echo "  Expected: $EXPECTED_HASH" >&2
    echo "  Actual:   $ACTUAL_HASH" >&2
    exit 1
  fi
  echo "Checksum verified."
fi

# Extract and install.
mkdir -p "${HELM_PLUGIN_DIR}/bin"
tar -xzf "${TMPDIR}/${PROJECT_NAME}.tar.gz" -C "${HELM_PLUGIN_DIR}/bin"
chmod +x "${HELM_PLUGIN_DIR}/bin/${PROJECT_NAME}"

echo "helm-map v${PLUGIN_VERSION} installed successfully."
