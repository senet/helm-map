#!/usr/bin/env bash
set -euo pipefail

PLUGIN_VERSION="0.1.1"
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

  ARCHIVE_NAME="${PROJECT_NAME}_${OS}_${ARCH}.tar.gz"

  echo "Downloading pre-built binary..."
  HTTP_CODE="$(curl -sSL -w '%{http_code}' -o "${TMPDIR}/${ARCHIVE_NAME}" "$DOWNLOAD_URL")"
  if [ "$HTTP_CODE" != "200" ]; then
    echo "Download failed (HTTP ${HTTP_CODE})." >&2
    return 1
  fi

  # Download checksums and verify.
  curl -sSL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL" 2>/dev/null || true
  if [ -f "${TMPDIR}/checksums.txt" ]; then
    EXPECTED_HASH="$(grep "${ARCHIVE_NAME}" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
    if [ -n "$EXPECTED_HASH" ]; then
      if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_HASH="$(sha256sum "${TMPDIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
      elif command -v shasum >/dev/null 2>&1; then
        ACTUAL_HASH="$(shasum -a 256 "${TMPDIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
      else
        echo "Warning: no sha256sum or shasum found, skipping checksum verification." >&2
        ACTUAL_HASH=""
      fi
      if [ -n "$ACTUAL_HASH" ] && [ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]; then
        echo "Checksum verification failed!" >&2
        echo "  Expected: $EXPECTED_HASH" >&2
        echo "  Actual:   $ACTUAL_HASH" >&2
        exit 1
      fi
      [ -n "$ACTUAL_HASH" ] && echo "Checksum verified."
    fi
  fi

  # Extract and install.
  mkdir -p "${HELM_PLUGIN_DIR}/bin"
  tar -xzf "${TMPDIR}/${ARCHIVE_NAME}" -C "${HELM_PLUGIN_DIR}/bin"
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
  echo "Pre-built binary not available, falling back to source build..."
  build_from_source
fi

echo "helm-map v${PLUGIN_VERSION} installed successfully."
