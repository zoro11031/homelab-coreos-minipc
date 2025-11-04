#!/bin/bash
# WireGuard Connection Health Check
# Monitors VPN connection to VPS and attempts to reconnect if needed

set -euo pipefail

# Configuration
INTERFACE="wg0"
VPS_ENDPOINT="10.99.0.1"  # VPS internal IP on WireGuard network
LOG_FILE="/var/log/wireguard-health.log"
ALERT_FILE="/var/run/wireguard-alert"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

check_interface() {
    if ! ip link show "$INTERFACE" &>/dev/null; then
        echo -e "${RED}✗${NC} WireGuard interface $INTERFACE does not exist"
        return 1
    fi

    if ! ip link show "$INTERFACE" | grep -q "state UP"; then
        echo -e "${RED}✗${NC} WireGuard interface $INTERFACE is down"
        return 1
    fi

    echo -e "${GREEN}✓${NC} WireGuard interface $INTERFACE is up"
    return 0
}

check_connectivity() {
    echo -n "Testing connectivity to VPS ($VPS_ENDPOINT)... "

    if ping -c 3 -W 2 "$VPS_ENDPOINT" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

show_wireguard_status() {
    echo
    echo "WireGuard Status:"
    echo "─────────────────────────────────────────────────────────────"
    wg show "$INTERFACE" 2>/dev/null || echo "Unable to retrieve WireGuard status"
    echo "─────────────────────────────────────────────────────────────"
}

attempt_reconnect() {
    log "WARN: Attempting to reconnect WireGuard"

    # Try to restart the service
    systemctl restart "wg-quick@$INTERFACE.service"
    sleep 5

    # Check if it's back up
    if check_interface && check_connectivity; then
        log "SUCCESS: WireGuard reconnected successfully"
        return 0
    else
        log "ERROR: Failed to reconnect WireGuard"
        return 1
    fi
}

check_bandwidth() {
    echo
    echo "Recent bandwidth usage:"
    if command -v vnstat &>/dev/null; then
        vnstat -i "$INTERFACE" -l 1 2>/dev/null || echo "vnstat not available"
    else
        echo "Install vnstat for bandwidth statistics"
    fi
}

main() {
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║  WireGuard Health Check - $(date '+%Y-%m-%d %H:%M:%S')       ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo

    healthy=true

    # Check systemd service
    echo -n "Checking systemd service... "
    if systemctl is-active --quiet "wg-quick@$INTERFACE.service"; then
        echo -e "${GREEN}✓${NC} Active"
    else
        echo -e "${RED}✗${NC} Inactive"
        healthy=false
    fi

    # Check interface
    if ! check_interface; then
        healthy=false
    fi

    # Check connectivity
    if ! check_connectivity; then
        healthy=false
    fi

    # Show detailed status if requested
    if [ "${VERBOSE:-false}" = "true" ] || [ "$healthy" = false ]; then
        show_wireguard_status
    fi

    echo

    if [ "$healthy" = true ]; then
        echo -e "${GREEN}WireGuard connection is healthy${NC}"
        rm -f "$ALERT_FILE"
        log "INFO: WireGuard connection healthy"

        # Show bandwidth if available
        if [ "${SHOW_BANDWIDTH:-false}" = "true" ]; then
            check_bandwidth
        fi

        return 0
    else
        echo -e "${RED}WireGuard connection is unhealthy${NC}"

        # Attempt automatic reconnect
        if [ "${AUTO_RECONNECT:-true}" = "true" ]; then
            echo
            if attempt_reconnect; then
                echo -e "${GREEN}Successfully reconnected${NC}"
                rm -f "$ALERT_FILE"
                return 0
            else
                echo -e "${RED}Failed to reconnect${NC}"
                touch "$ALERT_FILE"
            fi
        else
            touch "$ALERT_FILE"
        fi

        echo "Check logs: $LOG_FILE"
        log "ERROR: WireGuard health check failed"

        # Optional: Send notification
        # notify-send "WireGuard Connection Failure" "VPN to VPS is down"

        return 1
    fi
}

# If running as a systemd timer, be quiet
if [ "${QUIET:-false}" = "true" ]; then
    main >> "$LOG_FILE" 2>&1
else
    main
fi
