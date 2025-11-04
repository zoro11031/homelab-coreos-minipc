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

echo "Generating wg0.conf from template..."

python3 - "$TEMPLATE_FILE" "$OUTPUT_FILE" <<'PY'
import os
import sys
from pathlib import Path

template_path = Path(sys.argv[1])
output_path = Path(sys.argv[2])

placeholder_map = {
    "[SERVER_PRIVATE_KEY]": os.environ["WG_SERVER_PRIVATE_KEY"],
    "[DESKTOP_PUBLIC_KEY]": os.environ["WG_PEER_DESKTOP_PUBLIC_KEY"],
    "[DESKTOP_PRESHARED_KEY]": os.environ["WG_PEER_DESKTOP_PRESHARED_KEY"],
    "[VPS_PUBLIC_KEY]": os.environ["WG_PEER_VPS_PUBLIC_KEY"],
    "[VPS_PRESHARED_KEY]": os.environ["WG_PEER_VPS_PRESHARED_KEY"],
    "[IPHONE_PUBLIC_KEY]": os.environ["WG_PEER_IPHONE_PUBLIC_KEY"],
    "[IPHONE_PRESHARED_KEY]": os.environ["WG_PEER_IPHONE_PRESHARED_KEY"],
    "[LAPTOP_PUBLIC_KEY]": os.environ["WG_PEER_LAPTOP_PUBLIC_KEY"],
    "[LAPTOP_PRESHARED_KEY]": os.environ["WG_PEER_LAPTOP_PRESHARED_KEY"],
}

template = template_path.read_text()
missing_placeholders = [placeholder for placeholder in placeholder_map if placeholder not in template]

if missing_placeholders:
    print(
        "Error: Missing placeholders in template: "
        + ", ".join(sorted(missing_placeholders)),
        file=sys.stderr,
    )
    sys.exit(1)

for placeholder, value in placeholder_map.items():
    template = template.replace(placeholder, value)

output_path.write_text(template)
PY

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
