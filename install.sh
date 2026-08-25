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

# checksum_tool prints the sha256 command available on this machine.
checksum_tool() {
  if command -v shasum >/dev/null 2>&1; then
    echo "shasum -a 256"
  elif command -v sha256sum >/dev/null 2>&1; then
    echo "sha256sum"
  else
    echo "orbit: neither shasum nor sha256sum is on this machine — cannot verify the download" >&2
    exit 1
  fi
}

main() {
  local os arch version archive url checksums_url work_dir tool want got

  os=$(detect_os)
  arch=$(detect_arch)
  version=$(resolve_version)
  archive=$(archive_name "$version" "$os" "$arch")

  echo "orbit: installing ${version} for ${os}/${arch}"

  work_dir=$(mktemp -d)
  trap 'rm -rf "$work_dir"' EXIT

  url=$(download_url "$version" "$archive")
  checksums_url=$(download_url "$version" "checksums.txt")

  curl -fsSL "$url" -o "$work_dir/$archive"
  curl -fsSL "$checksums_url" -o "$work_dir/checksums.txt"

  tool=$(checksum_tool)
  want=$(grep "  ${archive}\$" "$work_dir/checksums.txt" | awk '{print $1}')
  if [ -z "$want" ]; then
    echo "orbit: ${archive} is not listed in checksums.txt — refusing to install" >&2
    exit 1
  fi
  got=$($tool "$work_dir/$archive" | awk '{print $1}')
  if [ "$want" != "$got" ]; then
    echo "orbit: checksum mismatch for ${archive} — refusing to install" >&2
    echo "  expected: $want" >&2
    echo "  got:      $got" >&2
    exit 1
  fi

  tar -xzf "$work_dir/$archive" -C "$work_dir"
  mkdir -p "$ORBIT_INSTALL_DIR"
  install -m 0755 "$work_dir/orbit" "$ORBIT_INSTALL_DIR/orbit"

  echo "orbit: installed ${version} to ${ORBIT_INSTALL_DIR}/orbit"

  case ":$PATH:" in
    *":$ORBIT_INSTALL_DIR:"*) ;;
    *)
      echo ""
      echo "orbit: ${ORBIT_INSTALL_DIR} is not on your PATH. Add this to your shell rc file:"
      echo "  export PATH=\"${ORBIT_INSTALL_DIR}:\$PATH\""
      ;;
  esac

  echo ""
  echo "Run 'orbit --help' to get started."

  # Exit explicitly (rather than falling off the end of the function)
  # so the EXIT trap above fires while work_dir is still in scope: a
  # trap set inside a function is process-wide, but the function's
  # `local` variables are torn down once it returns. If main() were
  # to just return here, the trap would fire afterward at top level,
  # where work_dir is unset — an unbound-variable error under `set
  # -u` on every successful install.
  exit 0
}

if [ "${BASH_SOURCE[0]:-$0}" = "${0}" ]; then
  main "$@"
fi
