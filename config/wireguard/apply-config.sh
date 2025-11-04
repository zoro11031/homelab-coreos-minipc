#!/bin/bash
# WireGuard Configuration Script
# This script applies the keys from .env to the wg0.conf template

set -e

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
    if [ -z "${!var}" ]; then
        echo -e "${RED}Error: $var is not set in .env${NC}"
        exit 1
    fi
done

echo "Generating wg0.conf from template..."

# Generate wg0.conf from template with all keys replaced
cat "$TEMPLATE_FILE" | \
    sed "s|\[PRIVATE KEY\]|$WG_SERVER_PRIVATE_KEY|g" | \
    awk -v desktop_pub="$WG_PEER_DESKTOP_PUBLIC_KEY" \
        -v desktop_psk="$WG_PEER_DESKTOP_PRESHARED_KEY" \
        -v vps_pub="$WG_PEER_VPS_PUBLIC_KEY" \
        -v vps_psk="$WG_PEER_VPS_PRESHARED_KEY" \
        -v iphone_pub="$WG_PEER_IPHONE_PUBLIC_KEY" \
        -v iphone_psk="$WG_PEER_IPHONE_PRESHARED_KEY" \
        -v laptop_pub="$WG_PEER_LAPTOP_PUBLIC_KEY" \
        -v laptop_psk="$WG_PEER_LAPTOP_PRESHARED_KEY" '
    {
        line = $0
        if (line ~ /^PublicKey=\[PUBLIC KEY\]/ && !desktop_done) {
            print "PublicKey=" desktop_pub
            desktop_done = 1
            next
        }
        if (line ~ /^PresharedKey=\[PRE SHARED KEY\]/ && !desktop_psk_done) {
            print "PresharedKey=" desktop_psk
            desktop_psk_done = 1
            next
        }
        if (line ~ /^PublicKey=\[PUBLIC KEY\]/ && desktop_done && !vps_done) {
            print "PublicKey=" vps_pub
            vps_done = 1
            next
        }
        if (line ~ /^PresharedKey=\[PRESHARED KEY\]/ && !vps_psk_done) {
            print "PresharedKey=" vps_psk
            vps_psk_done = 1
            next
        }
        if (line ~ /^PublicKey=\[PUBLIC KEY\]/ && vps_done && !iphone_done) {
            print "PublicKey=" iphone_pub
            iphone_done = 1
            next
        }
        if (line ~ /^PresharedKey=\[PRESHARED KEY\]/ && vps_psk_done && !iphone_psk_done) {
            print "PresharedKey=" iphone_psk
            iphone_psk_done = 1
            next
        }
        if (line ~ /^PublicKey=\[PUBLIC KEY\]/ && iphone_done && !laptop_done) {
            print "PublicKey=" laptop_pub
            laptop_done = 1
            next
        }
        if (line ~ /^PresharedKey=\[PRE SHARED KEY\]/ && iphone_psk_done && !laptop_psk_done) {
            print "PresharedKey=" laptop_psk
            laptop_psk_done = 1
            next
        }
        print line
    }' > "$OUTPUT_FILE"

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
