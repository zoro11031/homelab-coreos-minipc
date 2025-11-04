#!/bin/bash
# Generate client WireGuard configuration files for each peer

set -euo pipefail

usage() {
    cat <<'USAGE'
Usage: ./export-peer-configs.sh --endpoint <host:port> [options]

Options:
  --endpoint <host:port>   Required. Public endpoint for the WireGuard server.
  --allowed-ips <cidrs>    Comma-separated AllowedIPs for the peer. Default: 10.253.0.0/24
  --dns <resolver>         Optional DNS server pushed to clients (e.g., 1.1.1.1).
  --output-dir <path>      Directory to write client configs. Default: ./peer-configs
  --help                   Show this message and exit.
USAGE
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEYS_DIR="$SCRIPT_DIR/keys"
DEFAULT_OUTPUT_DIR="$SCRIPT_DIR/peer-configs"

ENDPOINT=""
ALLOWED_IPS="10.253.0.0/24"
DNS=""
OUTPUT_DIR="$DEFAULT_OUTPUT_DIR"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --endpoint)
            [[ $# -lt 2 ]] && { echo "Error: --endpoint requires a value" >&2; exit 1; }
            ENDPOINT="$2"
            shift 2
            ;;
        --allowed-ips)
            [[ $# -lt 2 ]] && { echo "Error: --allowed-ips requires a value" >&2; exit 1; }
            ALLOWED_IPS="$2"
            shift 2
            ;;
        --dns)
            [[ $# -lt 2 ]] && { echo "Error: --dns requires a value" >&2; exit 1; }
            DNS="$2"
            shift 2
            ;;
        --output-dir)
            [[ $# -lt 2 ]] && { echo "Error: --output-dir requires a value" >&2; exit 1; }
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Error: Unknown option $1" >&2
            usage
            exit 1
            ;;
    esac
done

if [[ -z "$ENDPOINT" ]]; then
    echo "Error: --endpoint is required" >&2
    usage
    exit 1
fi

if [[ ! -d "$KEYS_DIR" ]]; then
    echo "Error: $KEYS_DIR directory not found. Run ./generate-keys.sh first." >&2
    exit 1
fi

SERVER_PUBLIC_KEY_FILE="$KEYS_DIR/server-public.key"
if [[ ! -f "$SERVER_PUBLIC_KEY_FILE" ]]; then
    echo "Error: Server public key not found at $SERVER_PUBLIC_KEY_FILE" >&2
    exit 1
fi

SERVER_PUBLIC_KEY="$(<"$SERVER_PUBLIC_KEY_FILE")"

umask 077
mkdir -p "$OUTPUT_DIR"

declare -A PEER_LABELS=(
    [desktop]="LAN-Desktop-Justin"
    [vps]="VPS"
    [iphone]="iPhone"
    [laptop]="Framework Laptop Justin"
)

declare -A PEER_ADDRESSES=(
    [desktop]="10.253.0.6/32"
    [vps]="10.253.0.8/32"
    [iphone]="10.253.0.9/32"
    [laptop]="10.253.0.11/32"
)

PEER_ORDER=(desktop vps iphone laptop)

for peer in "${PEER_ORDER[@]}"; do
    peer_private_key_file="$KEYS_DIR/${peer}-private.key"
    peer_preshared_key_file="$KEYS_DIR/${peer}-preshared.key"

    if [[ ! -f "$peer_private_key_file" ]]; then
        echo "Error: Missing private key for ${PEER_LABELS[$peer]} ($peer_private_key_file)" >&2
        exit 1
    fi

    if [[ ! -f "$peer_preshared_key_file" ]]; then
        echo "Error: Missing preshared key for ${PEER_LABELS[$peer]} ($peer_preshared_key_file)" >&2
        exit 1
    fi

    peer_private_key="$(<"$peer_private_key_file")"
    peer_preshared_key="$(<"$peer_preshared_key_file")"
    peer_address="${PEER_ADDRESSES[$peer]}"

    if [[ -z "$peer_address" ]]; then
        echo "Error: No address configured for peer key '$peer'" >&2
        exit 1
    fi

    output_file="$OUTPUT_DIR/${peer}.conf"

    {
        echo "# ${PEER_LABELS[$peer]}"
        echo "[Interface]"
        echo "PrivateKey=$peer_private_key"
        echo "Address=$peer_address"
        [[ -n "$DNS" ]] && echo "DNS=$DNS"
        echo
        echo "[Peer]"
        echo "PublicKey=$SERVER_PUBLIC_KEY"
        echo "PresharedKey=$peer_preshared_key"
        echo "Endpoint=$ENDPOINT"
        echo "AllowedIPs=$ALLOWED_IPS"
        echo "PersistentKeepalive=30"
    } > "$output_file"

done

echo "Generated client configs in $OUTPUT_DIR:" >&2
for peer in "${PEER_ORDER[@]}"; do
    echo "  - $OUTPUT_DIR/${peer}.conf" >&2
done
