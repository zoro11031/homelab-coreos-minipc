# Setup Guide - NAB9 Mini PC

Complete setup guide for the NAB9 mini PC running uCore (Ublue CoreOS) as a frontend application node.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Initial OS Installation](#initial-os-installation)
3. [Repository Configuration](#repository-configuration)
4. [Network Configuration](#network-configuration)
5. [Service Deployment](#service-deployment)
6. [Verification](#verification)
7. [Next Steps](#next-steps)

## Prerequisites

### Hardware Requirements

- NAB9 Mini PC with Intel 12th gen+ CPU (QuickSync support)
- 16GB+ RAM
- 256GB+ NVMe SSD
- 2.5GbE ethernet connection
- IP KVM for remote BIOS access (recommended)

### Network Requirements

- Static IP address on local network (192.168.7.x)
- Access to NFS file server (192.168.7.10)
- DigitalOcean VPS with WireGuard configured
- Port forwarding for Plex (32400) and Jellyfin (8096, 8920)

### Software Requirements

- uCore (Ublue CoreOS) ISO image
- USB drive for installation (8GB+)
- SSH client for remote access

## Initial OS Installation

### 1. Download uCore Image

Visit [Ublue website](https://universal-blue.org/) and download the latest uCore image:

```bash
# Download the ISO (on your local machine)
curl -LO https://github.com/ublue-os/ucore/releases/latest/download/ucore.iso

# Write to USB drive (replace /dev/sdX with your USB device)
sudo dd if=ucore.iso of=/dev/sdX bs=4M status=progress && sync
```

### 2. Boot and Install

1. Insert USB drive into NAB9 mini PC
2. Access BIOS (usually F2 or DEL key during boot)
3. Set boot order to USB first
4. Save and reboot
5. Follow the Anaconda installer:
   - Select installation destination (NVMe SSD)
   - Configure network (set static IP: 192.168.7.x)
   - Set hostname: `nab9-minipc`
   - Create admin user with SSH key
   - Begin installation

### 3. Post-Installation Setup

After installation completes, reboot and remove the USB drive.

```bash
# SSH into the system
ssh your-user@192.168.7.x

# Update the system
sudo rpm-ostree upgrade
sudo systemctl reboot
```

## Repository Configuration

### 1. Clone This Repository

```bash
cd ~
git clone https://github.com/yourusername/homelab-coreos-minipc.git
cd homelab-coreos-minipc
```

### 2. Configure WireGuard

```bash
# Copy template and edit
cd config/wireguard
cp wg0.conf.template wg0.conf
nano wg0.conf

# Fill in:
# - Your private key
# - VPS public key
# - VPS endpoint IP
# - Preshared key (optional but recommended)

# Secure the file
chmod 600 wg0.conf
```

Generate WireGuard keys if you haven't already:

```bash
# On the NAB9 (client)
wg genkey | tee privatekey | wg pubkey > publickey

# On the VPS (server)
wg genkey | tee privatekey | wg pubkey > publickey

# Optional: Generate preshared key
wg genpsk > preshared.key
```

### 3. Configure NFS Mounts

Edit the NFS mount files to match your file server's IP and export paths:

```bash
# Check your file server's exports
showmount -e 192.168.7.10

# Edit mount units if needed
sudo nano files/system/etc/systemd/system/mnt-nas-media.mount
sudo nano files/system/etc/systemd/system/mnt-nas-nextcloud.mount
sudo nano files/system/etc/systemd/system/mnt-nas-immich.mount

# Update the "What=" line to match your server's exports
```

### 4. Configure Docker Compose

```bash
cd compose/
cp .env.example .env
nano .env

# Required configurations:
# - PLEX_CLAIM_TOKEN: Get from https://www.plex.tv/claim/
# - IMMICH_DB_PASSWORD: Generate strong password
# - TZ: Your timezone
# - APPDATA_PATH: Defaults to /var/lib/containers/appdata
```

### 5. Run Setup Script

```bash
cd ~/homelab-coreos-minipc
sudo ./scripts/setup.sh
```

This script will:
- Install WireGuard configuration
- Set up NFS mounts
- Configure firewall
- Set up GPU for hardware transcoding
- Create application directories
- Enable required services

## Network Configuration

### 1. Verify WireGuard Connection

```bash
# Check service status
sudo systemctl status wg-quick@wg0.service

# Show WireGuard interface details
sudo wg show

# Test connectivity to VPS
ping 10.99.0.1
```

### 2. Verify NFS Mounts

```bash
# Check mount status
systemctl status mnt-nas-media.mount
systemctl status mnt-nas-nextcloud.mount
systemctl status mnt-nas-immich.mount

# List mounted filesystems
df -h | grep nfs

# Test read access
ls -la /mnt/nas-media
ls -la /mnt/nas-nextcloud
ls -la /mnt/nas-immich
```

### 3. Configure Firewall

The setup script already configured UFW. Verify:

```bash
sudo ufw status verbose
```

Expected output:
- Port 22 (SSH) - rate limited
- Port 32400 (Plex)
- Ports 8096, 8920 (Jellyfin)
- Port 51820 (WireGuard)
- Allow from 192.168.7.0/24
- Allow from 10.99.0.0/24

### 4. Configure Router Port Forwarding

Configure your router to forward these ports to the NAB9's IP:

| Service   | External Port | Internal IP    | Internal Port | Protocol |
|-----------|---------------|----------------|---------------|----------|
| Plex      | 32400         | 192.168.7.x    | 32400         | TCP      |
| Jellyfin  | 8096          | 192.168.7.x    | 8096          | TCP      |
| Jellyfin  | 8920          | 192.168.7.x    | 8920          | TCP      |

## Service Deployment

### 1. Verify GPU Setup

```bash
cd ~/homelab-coreos-minipc
./scripts/gpu-verify.sh
```

This tests Intel QuickSync hardware transcoding.

### 2. Start Services

Start all services:

```bash
cd ~/homelab-coreos-minipc/compose
docker compose -f media.yml -f web.yml -f cloud.yml up -d
```

Or start individual service groups:

```bash
# Media services only (Plex, Jellyfin)
docker compose -f media.yml up -d

# Web services (Overseerr, Wizarr)
docker compose -f web.yml up -d

# Cloud services (Nextcloud, Immich)
docker compose -f cloud.yml up -d
```

### 3. Monitor Startup

```bash
# Watch logs
docker compose -f media.yml logs -f

# Check container status
docker ps

# Check specific service logs
docker logs -f plex
docker logs -f jellyfin
```

## Verification

### 1. Test Local Access

From within the local network:

- Plex: http://192.168.7.x:32400/web
- Jellyfin: http://192.168.7.x:8096
- Overseerr: http://192.168.7.x:5055
- Nextcloud: http://192.168.7.x:8080
- Immich: http://192.168.7.x:2283

### 2. Test Remote Access

#### Direct Access (Plex/Jellyfin)
- Plex: https://app.plex.tv
- Jellyfin: http://your-public-ip:8096

#### VPS Reverse Proxy Access

Configure Nginx Proxy Manager on your VPS to proxy these services:

| Service   | Internal URL           | Public Domain            |
|-----------|------------------------|--------------------------|
| Overseerr | http://10.99.0.2:5055  | overseerr.yourdomain.com |
| Wizarr    | http://10.99.0.2:5690  | invite.yourdomain.com    |
| Nextcloud | http://10.99.0.2:11000 | cloud.yourdomain.com     |
| Immich    | http://10.99.0.2:2283  | photos.yourdomain.com    |

### 3. Test Hardware Transcoding

In Plex:
1. Settings → Transcoder
2. Enable "Use hardware acceleration when available"
3. Select "Intel QuickSync"

In Jellyfin:
1. Dashboard → Playback
2. Enable "Intel QuickSync"
3. Select H.264, HEVC codecs

Test by playing a video that requires transcoding and monitor GPU usage:
```bash
intel_gpu_top
```

## Next Steps

1. **Configure Service Settings**
   - Set up Plex libraries pointing to /media
   - Configure Jellyfin libraries
   - Connect Overseerr to Plex/Jellyfin
   - Set up Nextcloud admin account
   - Configure Immich mobile app

2. **Set Up Monitoring**
   ```bash
   # Add health check timers
   sudo cp ~/homelab-coreos-minipc/config/systemd/nfs-health.timer /etc/systemd/system/
   sudo systemctl enable --now nfs-health.timer

   sudo cp ~/homelab-coreos-minipc/config/systemd/wireguard-health.timer /etc/systemd/system/
   sudo systemctl enable --now wireguard-health.timer
   ```

3. **Configure Automatic Updates**
   - Enable Watchtower for container updates
   - Set up rpm-ostree automatic updates

4. **Backup Configuration**
   - Back up Docker Compose .env file
   - Back up WireGuard configuration
   - Document VPS reverse proxy configuration

5. **Security Hardening**
   - Enable fail2ban
   - Set up SSH key-only authentication
   - Configure Cloudflare for DDoS protection
   - Enable two-factor authentication on all services

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues and solutions.

## References

- [Ublue Documentation](https://universal-blue.org/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [WireGuard Documentation](https://www.wireguard.com/)
- [Plex Hardware Transcoding](https://support.plex.tv/articles/115002178853-using-hardware-accelerated-streaming/)
- [Jellyfin Hardware Acceleration](https://jellyfin.org/docs/general/administration/hardware-acceleration.html)
