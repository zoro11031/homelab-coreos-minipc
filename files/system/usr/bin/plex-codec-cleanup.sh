#!/bin/bash
set -euo pipefail

# Plex Codec Cleanup Script
# Stops Plex, deletes codec directory, restarts container
# Prevents recurring EasyAudioEncoder corruption issues

echo "[$(date)] Starting Plex codec cleanup..."

# Stop Plex container
echo "Stopping Plex container..."
/usr/bin/docker stop plex || {
    echo "ERROR: Failed to stop Plex container"
    exit 1
}

# Wait for graceful shutdown
echo "Waiting 10 seconds for graceful shutdown..."
sleep 10

# Delete Codecs directory (entire directory, not just contents)
echo "Deleting Codecs directory..."
rm -rf "/var/lib/containers/appdata/plex/config/Library/Application Support/Plex Media Server/Codecs" || {
    echo "WARNING: Failed to delete Codecs directory (may not exist)"
}

# Wait before restart
echo "Waiting 5 seconds before restart..."
sleep 5

# Start Plex container
echo "Starting Plex container..."
/usr/bin/docker start plex || {
    echo "ERROR: Failed to start Plex container"
    exit 1
}

echo "[$(date)] Plex codec cleanup completed successfully"
