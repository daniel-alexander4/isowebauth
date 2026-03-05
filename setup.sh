#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GOPATH_BIN="$(go env GOPATH)/bin"
SHELL_RC=""

# Detect shell rc file
if [[ -n "${ZSH_VERSION:-}" ]] || [[ "$SHELL" == */zsh ]]; then
  SHELL_RC="$HOME/.zshrc"
else
  SHELL_RC="$HOME/.bashrc"
fi

echo "==> Checking Go installation..."
go version

# Ensure GOPATH/bin is in PATH persistently
if [[ ":$PATH:" != *":$GOPATH_BIN:"* ]]; then
  echo "==> Adding $GOPATH_BIN to PATH in $SHELL_RC"
  echo "" >> "$SHELL_RC"
  echo "# Go bin path (added by isowebauth desktop setup)" >> "$SHELL_RC"
  echo "export PATH=\"\$PATH:$(go env GOPATH)/bin\"" >> "$SHELL_RC"
  export PATH="$PATH:$GOPATH_BIN"
  echo "    Added. Will persist for new shells."
else
  echo "==> $GOPATH_BIN already in PATH"
fi

# Install system dependencies (Linux only)
if [[ "$(uname)" == "Linux" ]]; then
  echo "==> Checking Linux system dependencies..."
  MISSING=()
  dpkg -s libgtk-3-dev &>/dev/null || MISSING+=(libgtk-3-dev)
  dpkg -s libwebkit2gtk-4.0-dev &>/dev/null || MISSING+=(libwebkit2gtk-4.0-dev)

  if [[ ${#MISSING[@]} -gt 0 ]]; then
    echo "    Installing: ${MISSING[*]}"
    sudo apt-get update -qq
    sudo apt-get install -y -qq "${MISSING[@]}"
  else
    echo "    All system dependencies present."
  fi
fi

# Install Wails CLI
if command -v wails &>/dev/null; then
  echo "==> Wails CLI already installed: $(wails version 2>/dev/null || echo 'unknown')"
else
  echo "==> Installing Wails CLI..."
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  echo "    Installed: $(wails version 2>/dev/null || echo 'done')"
fi

# Run wails doctor
echo "==> Running wails doctor..."
wails doctor || true

# Install Go dependencies
echo "==> Installing Go dependencies..."
cd "$SCRIPT_DIR"
go mod tidy

# Run tests
echo "==> Running tests..."
go test ./internal/... -count=1
echo "    All tests passed."

# Build
echo "==> Building desktop app..."
wails build

echo ""
echo "==> Build complete!"
echo "    Binary: $SCRIPT_DIR/build/bin/isowebauth"
echo "    Run it:  ./build/bin/isowebauth"
