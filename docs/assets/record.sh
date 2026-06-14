#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ASSETS_DIR="$REPO_ROOT/docs/assets"
BIN_DIR="$ASSETS_DIR/.rec-bin"
HOME_DIR="$ASSETS_DIR/.rec-home"
MCP_WIRE="$REPO_ROOT/bin/mcp-wire"

rm -rf "$BIN_DIR" "$HOME_DIR"
mkdir -p "$BIN_DIR" "$HOME_DIR"

# Fake target binaries so the TUI sees all targets as installed.
cat > "$BIN_DIR/claude" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod +x "$BIN_DIR/claude"

cat > "$BIN_DIR/codex" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod +x "$BIN_DIR/codex"

cat > "$BIN_DIR/opencode" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod +x "$BIN_DIR/opencode"

# Pre-seed Claude Code config so uninstall has a service to remove.
mkdir -p "$HOME_DIR"
cat > "$HOME_DIR/.claude.json" <<'EOF'
{
  "mcpServers": {
    "playwright": {
      "type": "stdio",
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    }
  }
}
EOF

export PATH="$BIN_DIR:$PATH"
export HOME="$HOME_DIR"
export TERM="xterm-256color"
export MCP_WIRE="$MCP_WIRE"

record_cast() {
  local name="$1"
  shift
  local cast_file="$ASSETS_DIR/${name}.cast"
  local expect_script="$ASSETS_DIR/${name}.exp"

  echo "Recording ${name}..."
  rm -f "$cast_file"
  asciinema rec -q --cols 80 --rows 24 -c "expect ${expect_script}" "$cast_file"
}

cat > "$ASSETS_DIR/install.exp" <<'EOF'
set timeout 10
log_user 0
spawn env MCP_WIRE_REC=1 "$env(MCP_WIRE)"
expect "Install service"
log_user 1
sleep 0.5
send "\r"
expect "Search:"
sleep 0.5
send "playwright"
sleep 0.3
send "\r"
expect "Select targets:"
sleep 0.5
send " "
sleep 0.3
send "\r"
expect "Scope"
sleep 0.5
send "\r"
expect "Action:  Install service"
sleep 0.5
send "\r"
expect "installed successfully"
sleep 1.0
send "\033\[C"
sleep 0.2
send "\033\[C"
sleep 0.2
send "\r"
expect eof
EOF

cat > "$ASSETS_DIR/uninstall.exp" <<'EOF'
set timeout 10
log_user 0
spawn env MCP_WIRE_REC=1 "$env(MCP_WIRE)"
expect "Install service"
log_user 1
sleep 0.5
send "\033\[B"
sleep 0.3
send "\r"
expect "Select targets:"
sleep 0.5
send " "
sleep 0.3
send "\r"
expect "Search:"
sleep 0.5
send "\r"
expect "Scope"
sleep 0.5
send "\r"
expect "Action:  Uninstall service"
sleep 0.5
send "\r"
expect "removed successfully"
sleep 1.0
send "\033\[C"
sleep 0.2
send "\033\[C"
sleep 0.2
send "\r"
expect eof
EOF

record_cast install
record_cast uninstall

echo "Converting casts to SVG..."
asciinema convert -f asciicast-v2 --overwrite "$ASSETS_DIR/install.cast" "$ASSETS_DIR/install-v2.cast"
asciinema convert -f asciicast-v2 --overwrite "$ASSETS_DIR/uninstall.cast" "$ASSETS_DIR/uninstall-v2.cast"

svg-term --in "$ASSETS_DIR/install-v2.cast" --out "$ASSETS_DIR/install.svg" --width 80 --height 24 --window
svg-term --in "$ASSETS_DIR/uninstall-v2.cast" --out "$ASSETS_DIR/uninstall.svg" --width 80 --height 24 --window

rm "$ASSETS_DIR/install-v2.cast" "$ASSETS_DIR/uninstall-v2.cast"

echo "Done."
echo "Assets:"
ls -la "$ASSETS_DIR"/*.cast "$ASSETS_DIR"/*.svg

rm -rf "$BIN_DIR" "$HOME_DIR"
