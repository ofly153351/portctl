#!/bin/sh
set -eu

REPO=${PORTCTL_REPO:-ofly153351/portctl}
VERSION=${PORTCTL_VERSION:-latest}
INSTALL_DIR=${PORTCTL_INSTALL_DIR:-/usr/local/bin}

fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required to install portctl"

OS=$(uname -s)
ARCH=$(uname -m)
case "$OS" in
  Darwin) OS=darwin ;;
  Linux) OS=linux ;;
  *) fail "unsupported operating system: $OS (supported: macOS and Linux)" ;;
esac
case "$ARCH" in
  arm64|aarch64) ARCH=arm64 ;;
  x86_64|amd64) ARCH=amd64 ;;
  *) fail "unsupported architecture: $ARCH" ;;
esac

ASSET="portctl_${OS}_${ARCH}"
if [ "$VERSION" = latest ]; then
  URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
else
  case "$VERSION" in
    v*) TAG=$VERSION ;;
    *) TAG="v${VERSION}" ;;
  esac
  URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
fi

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/portctl-install.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT HUP INT TERM
BINARY="${TMP_DIR}/portctl"

printf 'Downloading %s (%s/%s)...\n' "$VERSION" "$OS" "$ARCH"
curl --fail --location --silent --show-error "$URL" -o "$BINARY" || fail "download failed; confirm that a published GitHub release has asset ${ASSET}"
chmod 755 "$BINARY"
mkdir -p "$INSTALL_DIR" || fail "could not create install directory ${INSTALL_DIR}"

if [ -w "$INSTALL_DIR" ]; then
  install -m 755 "$BINARY" "${INSTALL_DIR}/portctl" || fail "could not install to ${INSTALL_DIR}"
else
  command -v sudo >/dev/null 2>&1 || fail "${INSTALL_DIR} is not writable and sudo is unavailable; set PORTCTL_INSTALL_DIR to a writable directory"
  sudo install -m 755 "$BINARY" "${INSTALL_DIR}/portctl" || fail "could not install to ${INSTALL_DIR}"
fi

printf 'Installed portctl to %s/portctl\n' "$INSTALL_DIR"
printf 'Run portctl --help to get started.\n'
