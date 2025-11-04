#!/bin/bash
# WireGuard Configuration Script
# This script applies the keys from .env to the wg0.conf template

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
TEMPLATE_FILE="$SCRIPT_DIR/wg0.conf.template"
OUTPUT_FILE="$SCRIPT_DIR/wg0.conf"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== WireGuard Configuration Generator ===${NC}\n"

# Check if .env exists
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${RED}Error: $ENV_FILE not found${NC}"
    echo "Run ./generate-keys.sh first to generate keys"
    exit 1
fi

# Check if template exists
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo -e "${RED}Error: $TEMPLATE_FILE not found${NC}"
    exit 1
fi

# Load environment variables
echo "Loading keys from .env..."
set -a
source "$ENV_FILE"
set +a

# Verify all required variables are set
REQUIRED_VARS=(
    "WG_SERVER_PRIVATE_KEY"
    "WG_PEER_DESKTOP_PUBLIC_KEY"
    "WG_PEER_DESKTOP_PRESHARED_KEY"
    "WG_PEER_VPS_PUBLIC_KEY"
    "WG_PEER_VPS_PRESHARED_KEY"
    "WG_PEER_IPHONE_PUBLIC_KEY"
    "WG_PEER_IPHONE_PRESHARED_KEY"
    "WG_PEER_LAPTOP_PUBLIC_KEY"
    "WG_PEER_LAPTOP_PRESHARED_KEY"
)

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var:-}" ]; then
        echo -e "${RED}Error: $var is not set in .env${NC}"
        exit 1
    fi
done

echo "Validating template placeholders..."

PLACEHOLDERS=(
    "[SERVER_PRIVATE_KEY]"
    "[DESKTOP_PUBLIC_KEY]"
    "[DESKTOP_PRESHARED_KEY]"
    "[VPS_PUBLIC_KEY]"
    "[VPS_PRESHARED_KEY]"
    "[IPHONE_PUBLIC_KEY]"
    "[IPHONE_PRESHARED_KEY]"
    "[LAPTOP_PUBLIC_KEY]"
    "[LAPTOP_PRESHARED_KEY]"
)

for placeholder in "${PLACEHOLDERS[@]}"; do
    if ! grep -Fq "$placeholder" "$TEMPLATE_FILE"; then
        echo -e "${RED}Error: Placeholder $placeholder missing from template${NC}" >&2
        exit 1
    fi
done

echo "Generating wg0.conf from template..."

escape_sed_pattern() {
    printf '%s' "$1" | sed -e 's/[.[\\*^$]/\\&/g' -e 's/]/\\&/g'
}

escape_sed_replacement() {
    printf '%s' "$1" | sed -e 's/[\\&|]/\\&/g'
}

TEMP_FILE="${OUTPUT_FILE}.tmp"
cp "$TEMPLATE_FILE" "$TEMP_FILE"
trap 'rm -f "$TEMP_FILE"' EXIT

replace_placeholder() {
    local placeholder="$1"
    local value="$2"
    local escaped_placeholder
    local escaped_value

    escaped_placeholder="$(escape_sed_pattern "$placeholder")"
    escaped_value="$(escape_sed_replacement "$value")"

    if ! sed -i "s|${escaped_placeholder}|${escaped_value}|g" "$TEMP_FILE"; then
        echo -e "${RED}Error: Failed to replace placeholder ${placeholder}${NC}" >&2
        rm -f "$TEMP_FILE"
        exit 1
    fi
}

replace_placeholder "[SERVER_PRIVATE_KEY]" "$WG_SERVER_PRIVATE_KEY"
replace_placeholder "[DESKTOP_PUBLIC_KEY]" "$WG_PEER_DESKTOP_PUBLIC_KEY"
replace_placeholder "[DESKTOP_PRESHARED_KEY]" "$WG_PEER_DESKTOP_PRESHARED_KEY"
replace_placeholder "[VPS_PUBLIC_KEY]" "$WG_PEER_VPS_PUBLIC_KEY"
replace_placeholder "[VPS_PRESHARED_KEY]" "$WG_PEER_VPS_PRESHARED_KEY"
replace_placeholder "[IPHONE_PUBLIC_KEY]" "$WG_PEER_IPHONE_PUBLIC_KEY"
replace_placeholder "[IPHONE_PRESHARED_KEY]" "$WG_PEER_IPHONE_PRESHARED_KEY"
replace_placeholder "[LAPTOP_PUBLIC_KEY]" "$WG_PEER_LAPTOP_PUBLIC_KEY"
replace_placeholder "[LAPTOP_PRESHARED_KEY]" "$WG_PEER_LAPTOP_PRESHARED_KEY"

if grep -Eo '\[[A-Z_]+\]' "$TEMP_FILE" | grep -q "^\["; then
    echo -e "${RED}Error: Unresolved placeholders remain after substitution${NC}" >&2
    rm -f "$TEMP_FILE"
    exit 1
fi

mv "$TEMP_FILE" "$OUTPUT_FILE"
trap - EXIT
chmod 600 "$OUTPUT_FILE"

echo -e "\n${GREEN}=== Configuration Generated ===${NC}\n"
echo "Output file: $OUTPUT_FILE"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review the generated configuration:"
echo "   cat $OUTPUT_FILE"
echo ""
echo "2. Copy to system location:"
echo "   sudo cp $OUTPUT_FILE /etc/wireguard/wg0.conf"
echo "   sudo chmod 600 /etc/wireguard/wg0.conf"
echo ""
echo "3. Enable and start WireGuard:"
echo "   sudo systemctl enable --now wg-quick@wg0"
echo ""
echo "4. Check status:"
echo "   sudo wg show"
echo ""
echo -e "${RED}IMPORTANT: The generated wg0.conf contains sensitive keys!${NC}"
echo "Keep it secure and do not commit it to git."
