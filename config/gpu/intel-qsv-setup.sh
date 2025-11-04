#!/bin/bash
# Intel QuickSync GPU Setup Script for NAB9 Mini PC
# This script configures Intel QuickSync Video (QSV) for hardware transcoding
# in Plex, Jellyfin, and other media applications

set -euo pipefail

echo "=== Intel QuickSync GPU Setup ==="
echo

# Check if running on Intel CPU
if ! lscpu | grep -q "Intel"; then
    echo "ERROR: This system does not appear to have an Intel CPU"
    exit 1
fi

# Check for Intel GPU
echo "Checking for Intel GPU..."
if ! lspci | grep -i "VGA.*Intel"; then
    echo "WARNING: Intel GPU not detected via lspci"
    echo "This may be normal if the GPU is integrated"
fi

# Verify /dev/dri exists
echo
echo "Checking for /dev/dri devices..."
if [ ! -d "/dev/dri" ]; then
    echo "ERROR: /dev/dri directory does not exist"
    echo "Intel GPU drivers may not be loaded correctly"
    exit 1
fi

ls -l /dev/dri/
echo

# Check for render nodes
if [ ! -e "/dev/dri/renderD128" ]; then
    echo "WARNING: /dev/dri/renderD128 not found"
    echo "Hardware transcoding may not work properly"
fi

# Verify user permissions
echo "Checking user group memberships..."
CURRENT_USER="${SUDO_USER:-$USER}"
echo "Current user: $CURRENT_USER"

if ! groups "$CURRENT_USER" | grep -q "render"; then
    echo "Adding $CURRENT_USER to 'render' group for GPU access..."
    sudo usermod -a -G render "$CURRENT_USER"
    echo "User added to render group. You may need to log out and back in."
fi

if ! groups "$CURRENT_USER" | grep -q "video"; then
    echo "Adding $CURRENT_USER to 'video' group for GPU access..."
    sudo usermod -a -G video "$CURRENT_USER"
    echo "User added to video group. You may need to log out and back in."
fi

# Install VA-API tools if not present
echo
echo "Verifying VA-API tools..."
if ! command -v vainfo &> /dev/null; then
    echo "WARNING: vainfo not found. Install libva-utils package."
else
    echo "Running vainfo to check hardware acceleration capabilities..."
    echo "---"
    vainfo || echo "WARNING: vainfo failed to run"
    echo "---"
fi

# Check for Intel Media Driver
echo
echo "Checking for Intel Media Driver..."
if [ -f "/usr/lib64/dri/iHD_drv_video.so" ] || [ -f "/usr/lib/x86_64-linux-gnu/dri/iHD_drv_video.so" ]; then
    echo "✓ Intel Media Driver (iHD) found"
else
    echo "WARNING: Intel Media Driver not found. Install intel-media-driver package."
fi

# Check for i965 driver (older hardware)
if [ -f "/usr/lib64/dri/i965_drv_video.so" ] || [ -f "/usr/lib/x86_64-linux-gnu/dri/i965_drv_video.so" ]; then
    echo "✓ i965 driver found (for older Intel GPUs)"
fi

# Set environment variables for containers
echo
echo "=== Docker/Podman Environment Variables ==="
echo "Add these to your docker-compose.yml files:"
echo
echo "environment:"
echo "  - LIBVA_DRIVER_NAME=iHD"
echo "  - LIBVA_DRIVERS_PATH=/usr/lib/x86_64-linux-gnu/dri"
echo
echo "devices:"
echo "  - /dev/dri:/dev/dri"
echo

# Check kernel modules
echo "=== Kernel Modules ==="
echo "Checking for required kernel modules..."
for mod in i915 kvmgt vfio_iommu_type1 vfio_mdev; do
    if lsmod | grep -q "^$mod"; then
        echo "✓ $mod loaded"
    else
        echo "  $mod not loaded (may not be needed for basic QSV)"
    fi
done

echo
echo "=== Setup Complete ==="
echo "Hardware transcoding should now work with:"
echo "  - Plex"
echo "  - Jellyfin"
echo "  - Emby"
echo "  - FFmpeg-based applications"
echo
echo "IMPORTANT: If you added users to groups, log out and back in for changes to take effect."
