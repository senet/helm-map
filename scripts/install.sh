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

install_from_release() {
  TMPDIR="$(mktemp -d)"
  trap 'rm -rf "$TMPDIR"' EXIT

  # Download the binary archive and checksums.
  HTTP_CODE="$(curl -sSL -w '%{http_code}' "$DOWNLOAD_URL" -o "${TMPDIR}/${PROJECT_NAME}.tar.gz")"
  if [ "$HTTP_CODE" != "200" ]; then
    echo "Release download returned HTTP ${HTTP_CODE}." >&2
    return 1
  fi

  curl -sSL "$CHECKSUM_URL" -o "${TMPDIR}/checksums.txt" || true

  # Verify checksum if available.
  if [ -f "${TMPDIR}/checksums.txt" ]; then
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
  fi

  # Extract and install.
  mkdir -p "${HELM_PLUGIN_DIR}/bin"
  tar -xzf "${TMPDIR}/${PROJECT_NAME}.tar.gz" -C "${HELM_PLUGIN_DIR}/bin"
  chmod +x "${HELM_PLUGIN_DIR}/bin/${PROJECT_NAME}"
}

build_from_source() {
  echo "Building from source..."
  if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go is required to build from source. Install Go >= 1.22 and retry." >&2
    exit 1
  fi
  mkdir -p "${HELM_PLUGIN_DIR}/bin"
  cd "${HELM_PLUGIN_DIR}"
  go build -o "${HELM_PLUGIN_DIR}/bin/${PROJECT_NAME}" \
    -ldflags "-X main.version=${PLUGIN_VERSION}" \
    ./cmd/helm-map/
  echo "Built from source successfully."
}

if ! install_from_release; then
  echo "No pre-built release found, falling back to source build..."
  build_from_source
fi

echo "helm-map v${PLUGIN_VERSION} installed successfully."
