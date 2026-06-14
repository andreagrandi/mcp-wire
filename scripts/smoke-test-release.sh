#!/usr/bin/env bash
set -euo pipefail

resolve_binary() {
  if [[ $# -gt 0 && -n "${1:-}" ]]; then
    printf '%s\n' "$1"
    return 0
  fi

  if [[ -x "bin/mcp-wire" ]]; then
    printf '%s\n' "bin/mcp-wire"
    return 0
  fi

  if command -v mcp-wire >/dev/null 2>&1; then
    command -v mcp-wire
    return 0
  fi

  echo "error: no mcp-wire binary found (pass a path or build bin/mcp-wire)" >&2
  return 1
}

binary=$(resolve_binary "${1:-}")
homebrew_verify="${SMOKE_TEST_HOMEBREW:-false}"

if [[ ! -x "$binary" ]]; then
  echo "error: binary not found or not executable: $binary" >&2
  exit 1
fi

isolated_home=$(mktemp -d /tmp/mcp-wire-smoke.XXXXXX)
trap 'rm -rf "$isolated_home"' EXIT

export HOME="$isolated_home"
export PATH="/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin:$PATH"

echo "==> Smoke testing $binary with isolated HOME=$isolated_home"

version_output=$("$binary" --version)
echo "version: $version_output"
if [[ -z "$version_output" ]]; then
  echo "error: --version produced empty output" >&2
  exit 1
fi

help_output=$("$binary" --help)
if [[ "$help_output" != *"install and configure"* ]]; then
  echo "error: --help missing expected content" >&2
  exit 1
fi

doctor_output=$("$binary" doctor)
if [[ "$doctor_output" != *"Targets:"* ]]; then
  echo "error: doctor missing expected content" >&2
  exit 1
fi

metadata_output=$("$binary" metadata)
if [[ "$metadata_output" != *"\"schema_version\""* ]]; then
  echo "error: metadata missing expected content" >&2
  exit 1
fi

feature_list_output=$("$binary" feature list)
if [[ "$feature_list_output" != *"registry"* ]]; then
  echo "error: feature list missing expected content" >&2
  exit 1
fi

config_files=(
  "$isolated_home/.claude.json"
  "$isolated_home/.claude/settings.json"
  "$isolated_home/.codex/config.toml"
  "$isolated_home/.config/opencode/opencode.json"
  "$isolated_home/.config/mcp-wire/credentials"
  "$isolated_home/.config/mcp-wire/config.json"
)

for config in "${config_files[@]}"; do
  if [[ -e "$config" ]]; then
    echo "error: smoke test wrote unexpected config file: $config" >&2
    exit 1
  fi
done

echo "==> Basic smoke tests passed"

if [[ "$homebrew_verify" != "true" ]]; then
  exit 0
fi

if ! command -v brew >/dev/null 2>&1; then
  echo "warning: brew not found, skipping Homebrew verification" >&2
  exit 0
fi

if ! brew list mcp-wire >/dev/null 2>&1; then
  echo "warning: mcp-wire not installed via Homebrew, skipping Homebrew verification" >&2
  exit 0
fi

brew_prefix=$(brew --prefix mcp-wire)
brew_binary="$brew_prefix/bin/mcp-wire"
if [[ ! -x "$brew_binary" ]]; then
  echo "error: Homebrew-installed binary not found at $brew_binary" >&2
  exit 1
fi

"$brew_binary" --version
"$brew_binary" doctor >/dev/null
"$brew_binary" metadata >/dev/null
brew test mcp-wire

echo "==> Homebrew smoke tests passed"
