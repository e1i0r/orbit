#!/usr/bin/env bash
# install_test.sh exercises install.sh's pure logic — os/arch mapping,
# archive naming and URL construction — without touching the network.
# Run: bash install_test.sh

set -euo pipefail

# Source install.sh without running main: this file is not "$0" when
# sourced, so the guard at the bottom of install.sh skips it.
source "$(dirname "$0")/install.sh"

failures=0

assert_eq() {
  local desc="$1" want="$2" got="$3"
  if [ "$want" != "$got" ]; then
    echo "FAIL: $desc — want '$want', got '$got'"
    failures=$((failures + 1))
  else
    echo "ok: $desc"
  fi
}

assert_eq "os_name maps Darwin" "darwin" "$(os_name Darwin)"
assert_eq "os_name maps Linux" "linux" "$(os_name Linux)"
assert_eq "arch_name maps x86_64" "amd64" "$(arch_name x86_64)"
assert_eq "arch_name maps amd64" "amd64" "$(arch_name amd64)"
assert_eq "arch_name maps arm64" "arm64" "$(arch_name arm64)"
assert_eq "arch_name maps aarch64" "arm64" "$(arch_name aarch64)"

if os_name Windows >/dev/null 2>&1; then
  echo "FAIL: os_name should reject Windows"
  failures=$((failures + 1))
else
  echo "ok: os_name rejects Windows"
fi

if arch_name i386 >/dev/null 2>&1; then
  echo "FAIL: arch_name should reject i386"
  failures=$((failures + 1))
else
  echo "ok: arch_name rejects i386"
fi

assert_eq "archive_name strips the v prefix" \
  "orbit_0.1.5_darwin_arm64.tar.gz" \
  "$(archive_name v0.1.5 darwin arm64)"

assert_eq "download_url points at the release asset" \
  "https://github.com/e1i0r/orbit/releases/download/v0.1.5/orbit_0.1.5_darwin_arm64.tar.gz" \
  "$(download_url v0.1.5 orbit_0.1.5_darwin_arm64.tar.gz)"

if [ "$failures" -gt 0 ]; then
  echo "$failures failure(s)"
  exit 1
fi
echo "all checks passed"
