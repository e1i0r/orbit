#!/usr/bin/env bash
#
# Installs the latest orbit release for this machine's OS and
# architecture. Safe to re-run: it always fetches the current release
# and overwrites whatever is already installed.
#
# Usage:
#   curl -fsSL <url>/install.sh | bash
#
# Env overrides:
#   ORBIT_VERSION      pin to a specific tag (e.g. v0.1.4) instead of latest
#   ORBIT_INSTALL_DIR  where the binary goes (default: $HOME/.local/bin)
#   ORBIT_BASE_URL     where release assets are fetched from (for testing)

set -euo pipefail

ORBIT_REPO="${ORBIT_REPO:-e1i0r/orbit}"
ORBIT_INSTALL_DIR="${ORBIT_INSTALL_DIR:-$HOME/.local/bin}"
ORBIT_BASE_URL="${ORBIT_BASE_URL:-https://github.com/${ORBIT_REPO}/releases/download}"

# os_name maps a `uname -s` value to orbit's os name, or exits.
os_name() {
  case "$1" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *)
      echo "orbit: unsupported OS '$1' — only macOS and Linux have builds today" >&2
      return 1
      ;;
  esac
}

# arch_name maps a `uname -m` value to orbit's arch name, or exits.
arch_name() {
  case "$1" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "orbit: unsupported architecture '$1' — only amd64 and arm64 have builds today" >&2
      return 1
      ;;
  esac
}

detect_os() { os_name "$(uname -s)"; }
detect_arch() { arch_name "$(uname -m)"; }

# resolve_version prints the tag to install: $ORBIT_VERSION if the
# caller set one, otherwise the latest release's tag from GitHub.
resolve_version() {
  if [ -n "${ORBIT_VERSION:-}" ]; then
    echo "$ORBIT_VERSION"
    return
  fi
  local latest
  latest=$(curl -fsSL "https://api.github.com/repos/${ORBIT_REPO}/releases/latest" \
    | grep '"tag_name"' | head -n1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  if [ -z "$latest" ]; then
    echo "orbit: could not find a published release for ${ORBIT_REPO}" >&2
    return 1
  fi
  echo "$latest"
}

# archive_name prints the asset filename goreleaser publishes for a
# given version, os and arch — must match .goreleaser.yml's
# archives.name_template.
archive_name() {
  local version="$1" os="$2" arch="$3"
  echo "orbit_${version#v}_${os}_${arch}.tar.gz"
}

# download_url prints the full URL for one release asset.
download_url() {
  local version="$1" file="$2"
  echo "${ORBIT_BASE_URL}/${version}/${file}"
}

if [ "${BASH_SOURCE[0]:-$0}" = "${0}" ]; then
  echo "orbit: install.sh is not finished yet" >&2
  exit 1
fi
