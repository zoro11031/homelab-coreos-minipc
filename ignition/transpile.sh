#!/bin/bash

# Script to transpile Butane (.bu) configuration to Ignition (.ign) JSON
# Butane is a human-readable YAML format that gets converted to Ignition JSON

set -e

BUTANE_FILE="${1:-config.bu}"
IGNITION_FILE="${2:-config.ign}"

echo "CoreOS Butane to Ignition Transpiler"
echo "====================================="
echo ""

# Check if butane is available
if ! command -v butane &> /dev/null; then
    echo "Error: butane is not installed."
    echo ""
    echo "To install butane:"
    echo "  - Download from: https://github.com/coreos/butane/releases"
    echo "  - On Fedora: sudo dnf install butane"
    echo ""
    echo "For quick installation (Linux x86_64):"
    echo "  BUTANE_VERSION=v0.20.0"
    echo "  curl -L https://github.com/coreos/butane/releases/download/\${BUTANE_VERSION}/butane-x86_64-unknown-linux-gnu -o butane"
    echo "  chmod +x butane"
    echo "  sudo mv butane /usr/local/bin/"
    exit 1
fi

# Check if input file exists
if [ ! -f "$BUTANE_FILE" ]; then
    echo "Error: Butane file '$BUTANE_FILE' not found!"
    echo ""
    echo "Usage: $0 [input.bu] [output.ign]"
    echo "Example: $0 config.bu config.ign"
    exit 1
fi

# Validate that the file has been customized
if grep -q "YOUR_GOOD_PASSWORD_HASH_HERE" "$BUTANE_FILE"; then
    echo "Error: Please customize your Butane file first!"
    echo ""
    echo "You need to replace 'YOUR_GOOD_PASSWORD_HASH_HERE' with an actual password hash."
    echo "Run './generate-password-hash.sh' to generate a password hash."
    exit 1
fi

if grep -q "YOUR_SSH_PUB_KEY_HERE" "$BUTANE_FILE"; then
    echo "Error: Please customize your Butane file first!"
    echo ""
    echo "You need to replace 'YOUR_SSH_PUB_KEY_HERE' with your actual SSH public key."
    echo "Get your key with: cat ~/.ssh/id_ed25519.pub"
    exit 1
fi

echo "Transpiling $BUTANE_FILE to $IGNITION_FILE..."
butane --pretty --strict < "$BUTANE_FILE" > "$IGNITION_FILE"

echo ""
echo "Success! Ignition file created: $IGNITION_FILE"
echo ""
echo "You can now use this file to install CoreOS:"
echo "  - For bare metal: provide the URL or path to this file during installation"
echo "  - For VMs: use the Ignition file with your virtualization platform"
echo "  - For ISO: embed it using: coreos-installer iso ignition embed -i $IGNITION_FILE image.iso"
