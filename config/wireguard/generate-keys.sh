#!/bin/bash
# WireGuard Key Generation Script
# This script generates all necessary keys for the WireGuard server and peers

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
KEYS_DIR="$SCRIPT_DIR/keys"

DEFAULT_INTERFACE=""
OUTBOUND_INTERFACE_VALUE=""
OUTBOUND_INTERFACE_COMMENT=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== WireGuard Key Generation ===${NC}\n"

# Check if wireguard-tools is installed
if ! command -v wg &> /dev/null; then
    echo -e "${RED}Error: wireguard-tools is not installed${NC}"
    echo "Install it with: sudo dnf install wireguard-tools"
    exit 1
fi

if command -v ip &> /dev/null; then
    DEFAULT_INTERFACE="$(ip -o route show to default 2>/dev/null | awk '{print $5; exit}')"
fi

if [ -n "$DEFAULT_INTERFACE" ]; then
    OUTBOUND_INTERFACE_VALUE="$DEFAULT_INTERFACE"
    OUTBOUND_INTERFACE_COMMENT="Detected default network interface. Confirm before use."
    echo -e "${GREEN}Detected default WAN interface:${NC} $OUTBOUND_INTERFACE_VALUE"
else
    OUTBOUND_INTERFACE_VALUE="REPLACE_WITH_INTERFACE"
    OUTBOUND_INTERFACE_COMMENT="Set to the network interface that provides WAN access (e.g., enp1s0)."
    echo -e "${YELLOW}Warning:${NC} Unable to detect default WAN interface. Update WG_OUTBOUND_INTERFACE in .env manually."
fi

# Create keys directory if it doesn't exist
mkdir -p "$KEYS_DIR"
chmod 700 "$KEYS_DIR"

# Warn if .env already exists
if [ -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}Warning: $ENV_FILE already exists${NC}"
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi
fi

echo -e "${GREEN}Generating keys...${NC}\n"

# Generate server keys
echo "Generating server keys..."
wg genkey | tee "$KEYS_DIR/server-private.key" | wg pubkey > "$KEYS_DIR/server-public.key"

# Generate peer keys
declare -A PEERS=(
    ["desktop"]="LAN-Desktop-Justin"
    ["vps"]="VPS"
    ["iphone"]="iPhone"
    ["laptop"]="Framework Laptop Justin"
)

for peer in "${!PEERS[@]}"; do
    echo "Generating keys for ${PEERS[$peer]}..."
    wg genkey | tee "$KEYS_DIR/${peer}-private.key" | wg pubkey > "$KEYS_DIR/${peer}-public.key"
    wg genpsk > "$KEYS_DIR/${peer}-preshared.key"
done

# Set restrictive permissions on all key files
chmod 600 "$KEYS_DIR"/*.key

echo -e "\n${GREEN}Creating .env file...${NC}"

# Create .env file
cat > "$ENV_FILE" << EOF
# WireGuard Server Configuration
# Generated on $(date)
# DO NOT commit this file to git!

# Network interface used for NAT (WAN)
# ${OUTBOUND_INTERFACE_COMMENT}
WG_OUTBOUND_INTERFACE=${OUTBOUND_INTERFACE_VALUE}

# Server private key
WG_SERVER_PRIVATE_KEY=$(cat "$KEYS_DIR/server-private.key")

# Peer: LAN-Desktop-Justin
WG_PEER_DESKTOP_PUBLIC_KEY=$(cat "$KEYS_DIR/desktop-public.key")
WG_PEER_DESKTOP_PRESHARED_KEY=$(cat "$KEYS_DIR/desktop-preshared.key")

# Peer: VPS
WG_PEER_VPS_PUBLIC_KEY=$(cat "$KEYS_DIR/vps-public.key")
WG_PEER_VPS_PRESHARED_KEY=$(cat "$KEYS_DIR/vps-preshared.key")

# Peer: iPhone
WG_PEER_IPHONE_PUBLIC_KEY=$(cat "$KEYS_DIR/iphone-public.key")
WG_PEER_IPHONE_PRESHARED_KEY=$(cat "$KEYS_DIR/iphone-preshared.key")

# Peer: Framework Laptop Justin
WG_PEER_LAPTOP_PUBLIC_KEY=$(cat "$KEYS_DIR/laptop-public.key")
WG_PEER_LAPTOP_PRESHARED_KEY=$(cat "$KEYS_DIR/laptop-preshared.key")
EOF

chmod 600 "$ENV_FILE"

echo -e "\n${GREEN}=== Key Generation Complete ===${NC}\n"
echo "Generated files:"
echo "  - .env (contains all keys)"
echo "  - keys/ directory (individual key files)"
echo ""
echo -e "${YELLOW}Server Public Key (share this with clients):${NC}"
cat "$KEYS_DIR/server-public.key"
echo ""
echo -e "${YELLOW}Client configurations:${NC}"
echo ""

# Display client private keys for configuration
for peer in "${!PEERS[@]}"; do
    echo -e "${GREEN}${PEERS[$peer]}:${NC}"
    echo "  Private Key: $(cat "$KEYS_DIR/${peer}-private.key")"
    echo "  Public Key:  $(cat "$KEYS_DIR/${peer}-public.key")"
    echo "  Preshared:   $(cat "$KEYS_DIR/${peer}-preshared.key")"
    echo ""
done

echo -e "${YELLOW}Next steps:${NC}"
echo "1. Run: ./apply-config.sh to generate wg0.conf from template"
echo "2. Review the generated wg0.conf file"
echo "3. Copy wg0.conf to /etc/wireguard/wg0.conf on your server"
echo "4. Enable WireGuard: sudo systemctl enable --now wg-quick@wg0"
echo ""
echo -e "${RED}IMPORTANT: Keep the .env and keys/ directory secure!${NC}"
echo "Add them to .gitignore to prevent accidental commits."
