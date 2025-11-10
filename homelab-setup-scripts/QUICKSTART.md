# UBlue uCore Homelab - Quick Start Guide

Get your homelab up and running in under 30 minutes!

## Prerequisites

- ✅ UBlue uCore installed
- ✅ User with sudo access
- ✅ Internet connection
- ✅ (Optional) NFS server with configured exports
- ✅ Required packages: podman, podman-compose, nfs-utils, wireguard-tools

## Installation

### 1. Get the Scripts

```bash
cd ~
# Extract or clone homelab-setup-scripts
cd homelab-setup-scripts
```

### 2. Run Setup

```bash
./homelab-setup.sh
```

### 3. Choose Setup Method

#### Option A: Complete Setup (Recommended for first time)
- Includes WireGuard VPN setup
- Takes ~15-30 minutes
- Follow interactive prompts

#### Option Q: Quick Setup (Skip WireGuard)
- Skips VPN configuration
- Takes ~10-20 minutes
- Perfect if you don't need VPN

### 4. Follow the Prompts

The setup will ask you for:

**User Setup:**
- User account to run containers (default: current user)
- Automatically detects UID/GID and timezone

**NFS Setup:**
- NFS server IP address (default: 192.168.7.10)
- Automatically creates and mounts shares

**WireGuard Setup** (if using Complete Setup):
- WAN interface (auto-detected)
- Listen port (default: 51820)
- Server IP (default: 10.253.0.1/24)

**Container Setup:**
- Plex claim token (get from https://plex.tv/claim)
- Nextcloud admin password
- Database passwords for Nextcloud and Immich
- Trusted domain names

## What Gets Installed

### Media Services
- **Plex** (port 32400) - Media server
- **Jellyfin** (port 8096) - Open-source media server
- **Tautulli** (port 8181) - Plex statistics

### Web Services
- **Overseerr** (port 5055) - Media request management
- **Wizarr** (port 5690) - User invitation system
- **Organizr** (port 9983) - Service dashboard
- **Homepage** (port 3000) - Modern dashboard

### Cloud Services
- **Nextcloud** (port 8080) - File sync and share
- **Collabora** (port 9980) - Online office suite
- **Immich** (port 2283) - Photo management

## After Installation

### Access Your Services

```bash
# Replace <your-ip> with your server's IP address

# Media
http://<your-ip>:32400/web    # Plex
http://<your-ip>:8096          # Jellyfin
http://<your-ip>:8181          # Tautulli

# Web
http://<your-ip>:5055          # Overseerr
http://<your-ip>:5690          # Wizarr
http://<your-ip>:9983          # Organizr
http://<your-ip>:3000          # Homepage

# Cloud
http://<your-ip>:8080          # Nextcloud
http://<your-ip>:9980          # Collabora
http://<your-ip>:2283          # Immich
```

### Verify Everything is Running

```bash
# Check systemd services
sudo systemctl status podman-compose-media.service
sudo systemctl status podman-compose-web.service
sudo systemctl status podman-compose-cloud.service

# Check containers
podman ps

# Check NFS mounts
mount | grep /mnt/nas
```

## Quick Commands

### Service Management

```bash
# Restart a service
sudo systemctl restart podman-compose-media.service

# View service logs
sudo journalctl -u podman-compose-media.service -f

# Stop a service
sudo systemctl stop podman-compose-media.service

# Start a service
sudo systemctl start podman-compose-media.service
```

### Container Management

```bash
# List running containers
podman ps

# View container logs
podman logs plex

# Restart a container
podman restart plex

# Enter container shell
podman exec -it plex /bin/bash
```

### NFS Mounts

```bash
# Check mount status
mount | grep /mnt/nas

# Remount NFS share
sudo systemctl restart mnt-nas-media.mount

# View mount logs
sudo journalctl -u mnt-nas-media.mount
```

### System Status

```bash
# Check rpm-ostree deployment
rpm-ostree status

# Update system
sudo rpm-ostree upgrade
sudo systemctl reboot
```

## Troubleshooting

### Services Not Starting?

```bash
# Run troubleshooting tool
./scripts/troubleshoot.sh

# Or check specific service
sudo systemctl status podman-compose-media.service
sudo journalctl -u podman-compose-media.service -n 50
```

### Containers Not Running?

```bash
# Check container status
podman ps -a

# View container logs
podman logs <container-name>

# Restart service
sudo systemctl restart podman-compose-<service>.service
```

### NFS Not Mounting?

```bash
# Test connectivity
ping <nfs-server-ip>

# Check exports
showmount -e <nfs-server-ip>

# View mount logs
sudo journalctl -u mnt-nas-media.mount

# Try manual mount
sudo mount -t nfs <server>:<export> /mnt/nas-media
```

### Permission Errors?

```bash
# Check ownership
ls -la /srv/containers
ls -la /var/lib/containers/appdata

# Fix ownership (replace <user> with your username)
sudo chown -R <user>:<user> /srv/containers
sudo chown -R <user>:<user> /var/lib/containers/appdata
```

### Out of Disk Space?

```bash
# Check disk usage
df -h

# Clean up old images
podman system prune -a

# Clean up old deployments
rpm-ostree cleanup -b
```

## Configuration Files

### Location

```bash
# Main configuration
~/.homelab-setup.conf

# Service configurations
/srv/containers/media/.env
/srv/containers/web/.env
/srv/containers/cloud/.env

# Compose files
/srv/containers/media/compose.yml
/srv/containers/web/compose.yml
/srv/containers/cloud/compose.yml
```

### Editing Configuration

```bash
# Edit environment variables
nano /srv/containers/media/.env

# Apply changes
sudo systemctl restart podman-compose-media.service
```

## Customization

### Adding More Services

1. Create new directory:
   ```bash
   mkdir -p /srv/containers/myservice
   ```

2. Create compose file:
   ```bash
   nano /srv/containers/myservice/compose.yml
   ```

3. Create environment file:
   ```bash
   nano /srv/containers/myservice/.env
   ```

4. Start with podman-compose:
   ```bash
   cd /srv/containers/myservice
   podman-compose up -d
   ```

### Modifying Existing Services

1. Edit compose file:
   ```bash
   nano /srv/containers/media/compose.yml
   ```

2. Restart service:
   ```bash
   sudo systemctl restart podman-compose-media.service
   ```

## Updating

### Update Container Images

```bash
# Update a specific service
cd /srv/containers/media
podman-compose pull
sudo systemctl restart podman-compose-media.service

# Update all services
for service in media web cloud; do
    cd /srv/containers/$service
    podman-compose pull
    sudo systemctl restart podman-compose-$service.service
done
```

### Update System

```bash
# Check for updates
rpm-ostree upgrade --check

# Apply updates
sudo rpm-ostree upgrade

# Reboot
sudo systemctl reboot
```

## Backup

### Quick Backup

```bash
# Backup compose files
tar -czf ~/homelab-compose-backup.tar.gz /srv/containers

# Backup configuration
cp ~/.homelab-setup.conf ~/.homelab-setup.conf.backup
```

### Full Backup (Application Data)

```bash
# Stop services first
sudo systemctl stop podman-compose-*.service

# Backup appdata (can be very large!)
tar -czf ~/homelab-appdata-backup.tar.gz /var/lib/containers/appdata

# Restart services
sudo systemctl start podman-compose-*.service
```

## Rerunning Setup

### Reconfigure a Specific Step

```bash
./homelab-setup.sh
# Select individual step number (0-6)
# Script will detect existing config and ask to reconfigure
```

### Complete Reset

```bash
./homelab-setup.sh
# Select: [R] Reset Setup
# This removes markers and backs up configuration
# Then run setup again
```

## Getting Help

### Built-in Troubleshooting

```bash
# Interactive menu
./scripts/troubleshoot.sh

# Run all diagnostics
./scripts/troubleshoot.sh --all

# Collect logs
./scripts/troubleshoot.sh --logs
```

### Check Documentation

```bash
# Read full README
cat README.md

# Or open in browser if available
```

### Common Issues

**"rpm-ostree not found"**
- You're not running on UBlue uCore
- These scripts are designed specifically for immutable Fedora

**"Package not found"**
- Install required packages:
  ```bash
  sudo rpm-ostree install podman podman-compose nfs-utils wireguard-tools
  sudo systemctl reboot
  ```

**"Permission denied"**
- Check sudo access: `sudo -v`
- Check file ownership in `/srv/containers` and `/var/lib/containers/appdata`

**"Cannot connect to Podman"**
- Start Podman socket: `systemctl --user start podman.socket`
- Or use: `sudo systemctl start podman.socket`

**"Templates not found"**
- Ensure home-directory-setup.service has run
- Check for templates in `~/setup/compose-setup` and `~/setup/wireguard-setup`
- Fallback: Templates should be in `/usr/share/compose-setup`

## Next Steps

1. **Configure Reverse Proxy** (optional)
   - Use Caddy, Nginx, or Traefik
   - Set up SSL certificates
   - Configure domain names

2. **Set Up Monitoring** (optional)
   - Add Prometheus and Grafana
   - Monitor system resources
   - Set up alerts

3. **Configure Backups**
   - Set up automated backups
   - Test restore procedures
   - Document backup strategy

4. **Secure Your Services**
   - Change default passwords
   - Enable two-factor authentication
   - Review firewall rules
   - Keep services updated

5. **Optimize Performance**
   - Adjust container resources
   - Configure hardware transcoding
   - Optimize database settings

## Tips and Tricks

### Monitor System Resources

```bash
# CPU and memory
htop

# Disk I/O
iotop

# Network
iftop

# Container stats
podman stats
```

### View All Logs

```bash
# All systemd services
sudo journalctl -xe

# All container logs
for container in $(podman ps --format "{{.Names}}"); do
    echo "=== $container ==="
    podman logs --tail 10 $container
done
```

### Quick Service Restart

```bash
# Restart all homelab services
sudo systemctl restart podman-compose-*.service
```

### Export WireGuard Configs

```bash
cd ~/setup/wireguard-setup
./export-peer-configs.sh
# Peer configs saved to peer-configs/
```

---

## Summary

✅ **Installation**: Run `./homelab-setup.sh` and follow prompts
✅ **Access**: Services available on various ports
✅ **Management**: Use `systemctl` for services, `podman` for containers
✅ **Troubleshooting**: Run `./scripts/troubleshoot.sh`
✅ **Help**: See README.md for detailed documentation

**Enjoy your homelab!** 🎉
