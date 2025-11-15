#!/usr/bin/env bash
#
# Installation script for homelab-setup Python package
#
# This script installs the homelab-setup package and its dependencies
# for the current user using pip.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing homelab-setup Python package..."

# Install package in user mode (no sudo required)
python3 -m pip install --user --upgrade pip

# Install the package
python3 -m pip install --user -e "${SCRIPT_DIR}"

echo "Installation complete!"
echo ""
echo "The 'homelab-setup' command is now available."
echo "Run 'homelab-setup --help' to see available commands."
echo ""
echo "Note: You may need to add ~/.local/bin to your PATH:"
echo "  export PATH=\$HOME/.local/bin:\$PATH"
