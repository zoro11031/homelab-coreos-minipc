# Intel QuickSync GPU Transcoding Guide

Complete guide to configuring and troubleshooting Intel QuickSync Video (QSV) hardware transcoding on the NAB9 mini PC.

## Table of Contents

1. [Overview](#overview)
2. [Hardware Requirements](#hardware-requirements)
3. [System Configuration](#system-configuration)
4. [Plex Configuration](#plex-configuration)
5. [Jellyfin Configuration](#jellyfin-configuration)
6. [Testing and Verification](#testing-and-verification)
7. [Troubleshooting](#troubleshooting)
8. [Performance Optimization](#performance-optimization)

## Overview

Intel QuickSync Video (QSV) is a hardware video encoding/decoding technology built into Intel CPUs. It significantly reduces CPU usage during transcoding while maintaining excellent video quality.

### Benefits

- **Lower CPU usage:** 10-30% CPU vs. 100% for software transcoding
- **More simultaneous streams:** Support 5-10+ transcodes simultaneously
- **Lower power consumption:** ~15-20W vs. 40-60W for CPU transcoding
- **Faster transcoding:** Real-time transcoding at higher resolutions
- **Better quality:** Hardware encoder optimized for quality/speed balance

### Supported Codecs

Intel 12th gen (Alder Lake) and newer support:

- **H.264 (AVC):** Decode and encode up to 4K
- **H.265 (HEVC):** Decode and encode up to 8K
- **VP9:** Decode up to 8K
- **AV1:** Decode only (encode on 11th gen Arc GPUs)

## Hardware Requirements

### CPU

- **Minimum:** Intel 7th gen (Kaby Lake) or newer
- **Recommended:** Intel 12th gen (Alder Lake) or newer
- **NAB9:** Intel 12th gen or 13th gen

### iGPU

The integrated GPU must be:
- **Enabled in BIOS**
- **Accessible in Linux** (appears as /dev/dri/renderD128)

### BIOS Settings

1. Enter BIOS (F2 or DEL during boot)
2. Locate graphics settings (usually under "Advanced" or "Chipset")
3. Set **Primary Display** to "Auto" or "IGFX"
4. Set **iGPU Multi-Monitor** to "Enabled" (if available)
5. Set **DVMT Pre-Allocated** to 64MB or higher
6. Save and reboot

## System Configuration

### 1. Verify Hardware Detection

Check if the GPU is detected:

```bash
# List PCI devices
lspci | grep -i vga

# Expected output (example):
# 00:02.0 VGA compatible controller: Intel Corporation Alder Lake-P Integrated Graphics Controller

# Check for /dev/dri devices
ls -l /dev/dri/

# Expected output:
# crw-rw----+ 1 root video  226,   0 Nov  4 12:00 card0
# crw-rw----+ 1 root render 226, 128 Nov  4 12:00 renderD128
```

### 2. Install Required Drivers

Drivers should already be installed via the Bluebuild recipe, but verify:

```bash
# Check for Intel media driver
rpm -qa | grep intel-media-driver

# Check for libva
rpm -qa | grep libva

# Install if missing
sudo rpm-ostree install intel-media-driver libva-utils mesa-dri-drivers
sudo systemctl reboot
```

### 3. Test VA-API

VA-API (Video Acceleration API) is the Linux interface for hardware video acceleration:

```bash
# Run vainfo
vainfo

# Expected output should include:
# VAProfileH264Main               : VAEntrypointVLD
# VAProfileH264Main               : VAEntrypointEncSlice
# VAProfileHEVCMain               : VAEntrypointVLD
# VAProfileHEVCMain               : VAEntrypointEncSlice
```

### 4. Set User Permissions

Users need to be in the `render` and `video` groups:

```bash
# Add your user to groups
sudo usermod -a -G render $USER
sudo usermod -a -G video $USER

# Log out and back in for changes to take effect

# Verify group membership
groups
```

### 5. Run Automated Setup

```bash
cd ~/homelab-coreos-minipc
sudo ./config/gpu/intel-qsv-setup.sh
```

## Plex Configuration

### 1. Enable Hardware Transcoding

1. Open Plex Web: http://192.168.7.x:32400/web
2. Go to Settings → Transcoder
3. Enable **"Use hardware acceleration when available"**
4. Select **"Intel QuickSync"** from the dropdown

### 2. Transcode Settings

Recommended settings:

- **Transcoder quality:** Automatic
- **Transcoder temporary directory:** `/transcode` (local SSD)
- **Background transcoding:** 30 minutes
- **Transcode throttle buffer:** 60 seconds

### 3. Verify Docker GPU Access

```bash
# Check if Plex container can see GPU
docker exec plex ls -l /dev/dri

# Should show card0 and renderD128
```

### 4. Test Transcoding

1. Play a video that requires transcoding
2. Open Plex dashboard (Settings → Status → Now Playing)
3. Look for **(hw)** next to the transcode info
4. Monitor GPU usage:
   ```bash
   # Install intel_gpu_top if not present
   sudo dnf install intel-gpu-tools

   # Monitor GPU in real-time
   sudo intel_gpu_top
   ```

## Jellyfin Configuration

### 1. Enable Hardware Acceleration

1. Open Jellyfin: http://192.168.7.x:8096
2. Go to Dashboard → Playback
3. Select **Hardware acceleration:** "Intel QuickSync (QSV)"
4. Enable hardware decoding for:
   - H264
   - HEVC
   - MPEG2
   - VC1
   - VP9

### 2. Hardware Encoding

Enable hardware encoding for:
- **H.264:** ✓
- **HEVC:** ✓
- **Allow encoding in HEVC format:** ✓

### 3. Transcoding Settings

- **Enable VPP Tone mapping:** ✓ (for HDR content)
- **Tonemapping algorithm:** BT.2390
- **Tonemapping mode:** Max
- **Prefer OS native DXVA or VA-API hardware decoders:** ✓

### 4. Verify Environment Variables

The docker-compose.yml should have:

```yaml
environment:
  - LIBVA_DRIVER_NAME=iHD
  - LIBVA_DRIVERS_PATH=/usr/lib/x86_64-linux-gnu/dri
devices:
  - /dev/dri:/dev/dri
```

### 5. Test Transcoding

1. Play a video that requires transcoding
2. Open Dashboard → Playback (while video is playing)
3. Look for hardware transcoding indication
4. Monitor GPU:
   ```bash
   sudo intel_gpu_top
   ```

## Testing and Verification

### Automated Test Script

```bash
cd ~/homelab-coreos-minipc
./scripts/gpu-verify.sh
```

This script:
- Checks hardware detection
- Verifies VA-API functionality
- Tests FFmpeg QSV encoders
- Performs actual transcode test
- Checks Docker container access

### Manual FFmpeg Test

```bash
# Create test video
ffmpeg -f lavfi -i testsrc=duration=10:size=1920x1080:rate=30 \
  -c:v libx264 -pix_fmt yuv420p test_input.mp4

# Transcode using QSV
ffmpeg -hwaccel qsv -hwaccel_device /dev/dri/renderD128 \
  -i test_input.mp4 \
  -c:v h264_qsv -preset medium -b:v 5M \
  test_output.mp4

# Check if successful
ls -lh test_output.mp4
```

### Monitor GPU Usage

```bash
# Real-time GPU monitoring
sudo intel_gpu_top

# Sample output while transcoding:
#  Freq   IRQ  RC6   Power  IMC   Render  Video
#  1450   /s   0%    15W    0%    35%     85%
```

During transcoding, you should see:
- **Video:** 60-90% usage
- **Render:** 10-30% usage
- **Power:** 15-25W
- **Frequency:** Near max (1200-1500 MHz)

## Troubleshooting

### Issue: /dev/dri Not Found

**Symptoms:** `/dev/dri` directory doesn't exist

**Solutions:**
1. Check if iGPU is enabled in BIOS
2. Verify Intel GPU drivers are installed:
   ```bash
   lsmod | grep i915
   rpm -qa | grep intel-media-driver
   ```
3. Check kernel messages:
   ```bash
   sudo dmesg | grep i915
   sudo dmesg | grep drm
   ```

### Issue: Permission Denied

**Symptoms:** Can't access /dev/dri/renderD128

**Solutions:**
1. Add user to groups:
   ```bash
   sudo usermod -a -G render,video $USER
   newgrp render
   ```
2. Check file permissions:
   ```bash
   ls -l /dev/dri/
   # Should show: crw-rw----+ root render/video
   ```
3. Restart Docker:
   ```bash
   sudo systemctl restart docker
   ```

### Issue: VA-API Not Working

**Symptoms:** vainfo shows errors or no profiles

**Solutions:**
1. Set environment variable:
   ```bash
   export LIBVA_DRIVER_NAME=iHD
   vainfo
   ```
2. Try alternative driver:
   ```bash
   export LIBVA_DRIVER_NAME=i965
   vainfo
   ```
3. Reinstall drivers:
   ```bash
   sudo rpm-ostree install --force intel-media-driver
   sudo systemctl reboot
   ```

### Issue: Plex Not Using Hardware Transcoding

**Symptoms:** CPU at 100%, no **(hw)** indication

**Solutions:**
1. Verify Plex Pass subscription (required for HW transcoding)
2. Check Transcoder settings in Plex
3. Verify container has GPU access:
   ```bash
   docker exec plex ls -l /dev/dri
   docker exec plex vainfo
   ```
4. Check Plex logs:
   ```bash
   docker logs plex | grep -i transcode
   docker logs plex | grep -i "hardware"
   ```
5. Restart Plex:
   ```bash
   docker restart plex
   ```

### Issue: Jellyfin Fails to Transcode

**Symptoms:** Playback errors, CPU transcode fallback

**Solutions:**
1. Check hardware acceleration settings in Dashboard → Playback
2. Verify environment variables in docker-compose.yml
3. Test VA-API in container:
   ```bash
   docker exec jellyfin vainfo
   ```
4. Check Jellyfin logs:
   ```bash
   docker logs jellyfin | grep -i vaapi
   docker logs jellyfin | grep -i "hardware"
   ```
5. Try different VA-API driver:
   ```yaml
   environment:
     - LIBVA_DRIVER_NAME=i965  # Instead of iHD
   ```

### Issue: Poor Transcoding Quality

**Symptoms:** Artifacts, blocky video

**Solutions:**
1. Increase bitrate:
   - Plex: Settings → Transcoder → Transcoder quality
   - Jellyfin: Dashboard → Playback → Transcoding thread count
2. Use software transcoding for certain codecs
3. Check source video quality
4. Update Intel media driver

### Issue: GPU Overheating

**Symptoms:** Throttling, reduced performance

**Solutions:**
1. Monitor temperature:
   ```bash
   sudo intel_gpu_top
   sensors
   ```
2. Check case airflow
3. Clean dust from vents
4. Limit concurrent transcodes in Plex/Jellyfin
5. Reduce transcoding quality/bitrate

## Performance Optimization

### Transcode Temporary Directory

Use fast local storage for transcode temp files:

```yaml
# In docker-compose.yml
volumes:
  - /var/lib/containers/appdata/plex/transcode:/transcode
```

Use SSD, not NFS:
```bash
# Create on local SSD
sudo mkdir -p /var/lib/containers/appdata/plex/transcode
sudo chown 1000:1000 /var/lib/containers/appdata/plex/transcode
```

### Concurrent Transcodes

Recommended limits based on hardware:

| Hardware | H.264 1080p | H.265 1080p | H.265 4K |
|----------|-------------|-------------|----------|
| 12th gen | 10-15       | 6-8         | 3-4      |
| 13th gen | 12-18       | 8-10        | 4-5      |

Configure in Plex:
- Settings → Network → Maximum simultaneous video transcode

### Power Management

Disable aggressive power saving:

```bash
# Add kernel parameter
sudo nano /etc/default/grub
# Add: intel_idle.max_cstate=1

sudo grub2-mkconfig -o /boot/grub2/grub.cfg
sudo systemctl reboot
```

### Network Optimization

For remote streaming:
1. Configure remote bitrate limits in Plex
2. Use Cloudflare or CDN for better routing
3. Enable hardware transcoding to reduce latency

## Benchmarks

Expected performance with Intel 12th gen:

| Scenario | CPU Usage | GPU Usage | Power | Quality |
|----------|-----------|-----------|-------|---------|
| 1x 1080p H.264 transcode | 10-15% | 70-80% | 15-20W | Excellent |
| 3x 1080p H.264 transcode | 20-25% | 80-90% | 20-25W | Excellent |
| 1x 4K HEVC → 1080p | 15-20% | 75-85% | 20-25W | Good |
| 3x 4K HEVC → 1080p | 30-40% | 85-95% | 25-30W | Good |
| Software (no GPU) 1x 1080p | 80-100% | 0% | 40-60W | Excellent |

## References

- [Intel QuickSync Documentation](https://www.intel.com/content/www/us/en/products/docs/processors/core/quick-sync-video.html)
- [Plex Hardware Transcoding](https://support.plex.tv/articles/115002178853-using-hardware-accelerated-streaming/)
- [Jellyfin Hardware Acceleration](https://jellyfin.org/docs/general/administration/hardware-acceleration.html)
- [VA-API Documentation](https://github.com/intel/libva)
- [FFmpeg QSV Documentation](https://trac.ffmpeg.org/wiki/Hardware/QuickSync)
