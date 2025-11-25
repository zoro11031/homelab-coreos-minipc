#!/bin/bash
# /usr/bin/plex-codec-cleanup.sh
# Optimized by SRE to prevent Thermal Runaway

CONTAINER_NAME="plex"
# Path updated based on your cat output
CODEC_DIR="/var/lib/containers/appdata/plex/config/Library/Application Support/Plex Media Server/Codecs"
MAX_TEMP=75000  # 75°C
LOG_TAG="plex-cleanup"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"; }

# 1. Thermal Safety Check
# Do not start maintenance if CPU is already cooking
CURRENT_TEMP=$(cat /sys/class/thermal/thermal_zone*/temp | sort -nr | head -n1)

if [ "$CURRENT_TEMP" -gt "$MAX_TEMP" ]; then
    log "CRITICAL: System too hot ($((CURRENT_TEMP/1000))°C). Aborting to prevent thermal shutdown."
    exit 1
fi

log "Starting cleanup. Temp: $((CURRENT_TEMP/1000))°C"

# 2. Graceful Stop (Extended Timeout)
# Give Plex 45 seconds to release the iGPU driver politely
log "Stopping Plex (Timeout: 45s)..."
/usr/bin/docker stop -t 45 "$CONTAINER_NAME"

# 3. Verify Death
# Ensure it is actually stopped before we touch files
if [ "$(/usr/bin/docker inspect -f '{{.State.Running}}' $CONTAINER_NAME)" == "true" ]; then
    log "CRITICAL ERROR: Plex refused to stop. Aborting restart to prevent driver corruption."
    exit 1
fi

# 4. Cleanup
log "Container stopped. Cleaning codecs..."
if [ -d "$CODEC_DIR" ]; then
    rm -rf "$CODEC_DIR"
else
    log "WARNING: Codec directory not found, skipping delete."
fi

# 5. Restart
log "Restarting Plex..."
/usr/bin/docker start "$CONTAINER_NAME"
log "Maintenance complete."
