#!/bin/bash
# NFS Health Check Script
# Monitors NFS mounts and attempts to remount if they fail

set -euo pipefail

# Configuration
MOUNTS=(
    "mnt-nas-media.mount:/mnt/nas-media"
    "mnt-nas-nextcloud.mount:/mnt/nas-nextcloud"
    "mnt-nas-immich.mount:/mnt/nas-immich"
)

LOG_FILE="/var/log/nfs-health.log"
ALERT_FILE="/var/run/nfs-alert"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

check_mount() {
    local unit_name="$1"
    local mount_point="$2"

    # Check if systemd unit is active
    if ! systemctl is-active --quiet "$unit_name"; then
        echo -e "${RED}✗${NC} $mount_point (systemd unit inactive)"
        return 1
    fi

    # Check if mount point is actually mounted
    if ! mountpoint -q "$mount_point"; then
        echo -e "${RED}✗${NC} $mount_point (not mounted)"
        return 1
    fi

    # Check if we can read the mount
    if ! timeout 5 ls "$mount_point" >/dev/null 2>&1; then
        echo -e "${RED}✗${NC} $mount_point (unresponsive)"
        return 1
    fi

    echo -e "${GREEN}✓${NC} $mount_point (healthy)"
    return 0
}

attempt_remount() {
    local unit_name="$1"
    local mount_point="$2"

    log "WARN: Attempting to remount $mount_point"

    # Try to stop the mount
    systemctl stop "$unit_name" 2>/dev/null || true
    sleep 2

    # Try to start it again
    if systemctl start "$unit_name"; then
        sleep 2
        if mountpoint -q "$mount_point" && timeout 5 ls "$mount_point" >/dev/null 2>&1; then
            log "SUCCESS: Remounted $mount_point"
            return 0
        fi
    fi

    log "ERROR: Failed to remount $mount_point"
    return 1
}

main() {
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║  NFS Health Check - $(date '+%Y-%m-%d %H:%M:%S')              ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo

    all_healthy=true

    for mount_def in "${MOUNTS[@]}"; do
        unit_name="${mount_def%%:*}"
        mount_point="${mount_def##*:}"

        if ! check_mount "$unit_name" "$mount_point"; then
            all_healthy=false

            # Attempt automatic remount
            if [ "${AUTO_REMOUNT:-true}" = "true" ]; then
                if attempt_remount "$unit_name" "$mount_point"; then
                    all_healthy=true
                else
                    touch "$ALERT_FILE"
                fi
            else
                touch "$ALERT_FILE"
            fi
        fi
    done

    echo

    if [ "$all_healthy" = true ]; then
        echo -e "${GREEN}All NFS mounts are healthy${NC}"
        rm -f "$ALERT_FILE"
        log "INFO: All NFS mounts healthy"
        return 0
    else
        echo -e "${RED}One or more NFS mounts are unhealthy${NC}"
        echo "Check logs: $LOG_FILE"
        log "ERROR: NFS health check failed"

        # Optional: Send notification (configure with your notification system)
        # notify-send "NFS Mount Failure" "One or more NFS mounts are down"

        return 1
    fi
}

# If running as a systemd timer, be quiet
if [ "${QUIET:-false}" = "true" ]; then
    main >> "$LOG_FILE" 2>&1
else
    main
fi
