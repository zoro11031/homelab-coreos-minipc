#!/bin/bash
# GPU Transcoding Verification Script
# Tests Intel QuickSync hardware acceleration

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "╔════════════════════════════════════════════════════════════╗"
echo "║  Intel QuickSync GPU Verification                         ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo

# Check for Intel CPU
echo -n "Checking for Intel CPU... "
if lscpu | grep -q "Intel"; then
    echo -e "${GREEN}✓${NC}"
    lscpu | grep "Model name"
else
    echo -e "${RED}✗${NC}"
    echo "ERROR: No Intel CPU detected"
    exit 1
fi
echo

# Check for /dev/dri
echo -n "Checking for /dev/dri devices... "
if [ -d "/dev/dri" ]; then
    echo -e "${GREEN}✓${NC}"
    ls -l /dev/dri/
else
    echo -e "${RED}✗${NC}"
    echo "ERROR: /dev/dri not found"
    exit 1
fi
echo

# Check for render node
echo -n "Checking for render node... "
if [ -e "/dev/dri/renderD128" ]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠${NC} Not found (may still work)"
fi
echo

# Check VA-API
echo "Checking VA-API capabilities..."
if command -v vainfo &>/dev/null; then
    echo "───────────────────────────────────────────────────────────"
    vainfo 2>&1 | head -30
    echo "───────────────────────────────────────────────────────────"

    if vainfo 2>&1 | grep -q "VAProfileH264"; then
        echo -e "${GREEN}✓${NC} H.264 encoding supported"
    else
        echo -e "${YELLOW}⚠${NC} H.264 encoding may not be available"
    fi

    if vainfo 2>&1 | grep -q "VAProfileHEVC"; then
        echo -e "${GREEN}✓${NC} HEVC (H.265) encoding supported"
    else
        echo -e "${YELLOW}⚠${NC} HEVC encoding may not be available"
    fi
else
    echo -e "${YELLOW}⚠${NC} vainfo not found. Install libva-utils package."
fi
echo

# Check user permissions
echo "Checking user group memberships..."
CURRENT_USER="${SUDO_USER:-$USER}"
echo "Current user: $CURRENT_USER"

if groups "$CURRENT_USER" | grep -q "render"; then
    echo -e "${GREEN}✓${NC} User in 'render' group"
else
    echo -e "${YELLOW}⚠${NC} User not in 'render' group (may need this for GPU access)"
fi

if groups "$CURRENT_USER" | grep -q "video"; then
    echo -e "${GREEN}✓${NC} User in 'video' group"
else
    echo -e "${YELLOW}⚠${NC} User not in 'video' group (may need this for GPU access)"
fi
echo

# Test with ffmpeg if available
echo "Testing GPU transcoding with FFmpeg..."
if command -v ffmpeg &>/dev/null; then
    # Check for QSV encoders
    if ffmpeg -hide_banner -encoders 2>/dev/null | grep -q "h264_qsv"; then
        echo -e "${GREEN}✓${NC} FFmpeg has QSV H.264 encoder"
    else
        echo -e "${YELLOW}⚠${NC} FFmpeg QSV H.264 encoder not found"
    fi

    if ffmpeg -hide_banner -encoders 2>/dev/null | grep -q "hevc_qsv"; then
        echo -e "${GREEN}✓${NC} FFmpeg has QSV HEVC encoder"
    else
        echo -e "${YELLOW}⚠${NC} FFmpeg QSV HEVC encoder not found"
    fi

    # Create a test video and transcode it
    echo
    echo "Creating test video and attempting GPU transcode..."
    TEST_INPUT="/tmp/test_input.mp4"
    TEST_OUTPUT="/tmp/test_output.mp4"

    # Generate a short test video
    ffmpeg -hide_banner -loglevel warning \
        -f lavfi -i testsrc=duration=5:size=1920x1080:rate=30 \
        -c:v libx264 -pix_fmt yuv420p \
        "$TEST_INPUT" 2>/dev/null

    if [ -f "$TEST_INPUT" ]; then
        echo "Test input created: $TEST_INPUT"

        # Try to transcode using QSV
        echo "Attempting transcode with h264_qsv..."
        if timeout 30 ffmpeg -hide_banner -loglevel error \
            -hwaccel qsv -hwaccel_device /dev/dri/renderD128 \
            -i "$TEST_INPUT" \
            -c:v h264_qsv -preset medium -b:v 2M \
            "$TEST_OUTPUT" 2>/dev/null; then

            if [ -f "$TEST_OUTPUT" ] && [ -s "$TEST_OUTPUT" ]; then
                echo -e "${GREEN}✓ GPU transcoding test PASSED${NC}"
                echo "Output file created: $TEST_OUTPUT"
                ls -lh "$TEST_OUTPUT"
            else
                echo -e "${RED}✗ GPU transcoding test FAILED (output empty)${NC}"
            fi
        else
            echo -e "${RED}✗ GPU transcoding test FAILED${NC}"
            echo "Try running the GPU setup script: ./config/gpu/intel-qsv-setup.sh"
        fi

        # Cleanup
        rm -f "$TEST_INPUT" "$TEST_OUTPUT"
    else
        echo -e "${YELLOW}⚠${NC} Could not create test input"
    fi
else
    echo -e "${YELLOW}⚠${NC} FFmpeg not found, skipping transcode test"
fi
echo

# Check Docker container access
echo "Checking Docker container GPU access..."
if command -v docker &>/dev/null; then
    if docker ps &>/dev/null; then
        # Check if Plex or Jellyfin containers are running
        if docker ps --format '{{.Names}}' | grep -q "plex"; then
            echo -n "Testing Plex container GPU access... "
            if docker exec plex ls /dev/dri &>/dev/null; then
                echo -e "${GREEN}✓${NC}"
            else
                echo -e "${RED}✗${NC}"
            fi
        fi

        if docker ps --format '{{.Names}}' | grep -q "jellyfin"; then
            echo -n "Testing Jellyfin container GPU access... "
            if docker exec jellyfin ls /dev/dri &>/dev/null; then
                echo -e "${GREEN}✓${NC}"
            else
                echo -e "${RED}✗${NC}"
            fi
        fi

        if ! docker ps --format '{{.Names}}' | grep -qE "plex|jellyfin"; then
            echo "No media containers running. Start them to test GPU access."
        fi
    else
        echo "Docker is installed but not running"
    fi
else
    echo "Docker not found, skipping container tests"
fi
echo

echo "╔════════════════════════════════════════════════════════════╗"
echo "║  GPU Verification Complete                                ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo
echo "Summary:"
echo "  - If all checks passed, hardware transcoding should work"
echo "  - Enable hardware transcoding in Plex/Jellyfin settings"
echo "  - Monitor GPU usage: intel_gpu_top or watch -n 1 intel_gpu_top"
echo
