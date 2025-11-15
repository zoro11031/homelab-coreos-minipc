#!/usr/bin/env bash
#
# Install homelab-setup Python package during image build
#
# This script runs during the BlueBuild image build process to install
# the homelab-setup Python package system-wide.

set -euxo pipefail

PACKAGE_DIR="/tmp/homelab-setup-py"

if [[ ! -d "$PACKAGE_DIR" ]]; then
    echo "ERROR: Package directory not found: $PACKAGE_DIR"
    exit 1
fi

echo "Installing homelab-setup Python package..."

# Upgrade pip
python3 -m pip install --upgrade pip

# Install the package (system-wide during build)
python3 -m pip install "$PACKAGE_DIR"

echo "homelab-setup package installed successfully"

# Verify installation
if ! command -v homelab-setup &> /dev/null; then
    echo "WARNING: homelab-setup command not found in PATH"
    echo "Installed packages:"
    python3 -m pip list | grep homelab-setup || true
else
    echo "homelab-setup command is available"
    homelab-setup --version
fi
