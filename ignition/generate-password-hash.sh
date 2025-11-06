#!/bin/bash

# Script to generate a password hash for CoreOS Ignition
# This hash will be used for the core user's password

set -e

echo "CoreOS Password Hash Generator"
echo "=============================="
echo ""

# Check if mkpasswd is available
if command -v mkpasswd &> /dev/null; then
    echo "Enter password for the 'core' user:"
    read -s PASSWORD
    echo ""
    echo "Confirm password:"
    read -s PASSWORD_CONFIRM
    echo ""

    if [ "$PASSWORD" != "$PASSWORD_CONFIRM" ]; then
        echo "Error: Passwords do not match!"
        exit 1
    fi

    echo "Generating password hash..."
    HASH=$(mkpasswd --method=yescrypt --stdin <<< "$PASSWORD")
    echo ""
    echo "Password hash generated successfully:"
    echo "$HASH"
    echo ""
    echo "Copy this hash and replace 'YOUR_PASSWORD_HASH' in config.bu.template"
else
    echo "Error: mkpasswd is not installed."
    echo ""
    echo "To install mkpasswd:"
    echo "  - On Fedora/RHEL: sudo dnf install mkpasswd"
    echo "  - On Debian/Ubuntu: sudo apt install whois"
    echo "  - On macOS: brew install mkpasswd"
    echo ""
    echo "Alternatively, you can generate a password hash online or on another system:"
    echo "  mkpasswd --method=yescrypt"
    exit 1
fi
