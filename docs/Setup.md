# Setup Guide

This guide provides detailed instructions for setting up the homelab CoreOS mini PC after the initial installation.

## Table of Contents

- [Prerequisites](#prerequisites)
- [1. User Setup](#1-user-setup)
- [2. Directory Structure](#2-directory-structure)
- [3. WireGuard Configuration](#3-wireguard-configuration)
- [4. NFS Mounts Setup](#4-nfs-mounts-setup)
- [5. Container Setup](#5-container-setup)
  - [Option A: Podman Compose (Recommended)](#option-a-podman-compose-recommended)
  - [Option B: Docker](#option-b-docker)
- [6. Service Deployment](#6-service-deployment)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

- System successfully installed with Ignition configuration (see main README)
- SSH access to the system as the `core` user
- Network connectivity to your file server (if using NFS)
- Your WireGuard endpoint details (if setting up VPN)

---

## 1. User Setup

While you can run everything as the `core` user, it's recommended to create a dedicated user for managing containers.

### Create a Docker Management User

```bash
# Create a dedicated user for managing containers
sudo useradd -m -s /bin/bash dockeruser

# Add the user to necessary groups
sudo usermod -aG wheel dockeruser  # sudo access
sudo usermod -aG docker dockeruser # if using Docker

# Set a password for the new user
sudo passwd dockeruser
```

### Switch to the Docker User

```bash
# Switch to the new user
sudo su - dockeruser

# Or use the core user if preferred
# All remaining commands can be run as either user
```

---

## 2. Directory Structure

Set up a consistent directory structure for your container configurations and application data.

### Create Compose Directory Structure

```bash
# Create main compose directory
mkdir -p ~/compose

# Create subdirectories for different service stacks
mkdir -p ~/compose/media      # Media services (Plex, Jellyfin, etc.)
mkdir -p ~/compose/web        # Web services (Overseerr, Wizarr)
mkdir -p ~/compose/cloud      # Cloud services (Nextcloud, Immich)

# Alternative: Single directory approach
# mkdir -p ~/compose/services
```

### Create Application Data Directory

```bash
# Create appdata directory for persistent container data
mkdir -p ~/appdata

# Create subdirectories for each service
mkdir -p ~/appdata/plex
mkdir -p ~/appdata/jellyfin
mkdir -p ~/appdata/overseerr
mkdir -p ~/appdata/wizarr
mkdir -p ~/appdata/nextcloud
mkdir -p ~/appdata/immich
mkdir -p ~/appdata/postgres
mkdir -p ~/appdata/redis
```

### Recommended Directory Structure

```
/home/dockeruser/           (or /home/core/)
├── compose/                # Container orchestration files
│   ├── media/
│   │   ├── docker-compose.yml (or compose.yaml)
│   │   └── .env
│   ├── web/
│   │   ├── docker-compose.yml
│   │   └── .env
│   └── cloud/
│       ├── docker-compose.yml
│       └── .env
└── appdata/                # Application persistent data
    ├── plex/
    ├── jellyfin/
    ├── overseerr/
    ├── wizarr/
    ├── nextcloud/
    ├── immich/
    ├── postgres/
    └── redis/
```

### Alternative: Consolidated Structure

```
/home/dockeruser/
├── compose/
│   ├── docker-compose.yml  # All services in one file
│   └── .env
└── appdata/
    └── [service directories as above]
```

---

## 3. WireGuard Configuration

WireGuard provides secure VPN connectivity for remote access and VPS tunneling.

### Configuration Files Location

The WireGuard configuration lives in `/etc/wireguard/wg0.conf`. The image includes a template and helper scripts.

### Generate WireGuard Keys and Configuration

If you cloned this repository, you can use the provided scripts:

```bash
# Navigate to the wireguard setup directory
cd /usr/share/wireguard-setup  # If scripts are bundled
# OR clone the repo and navigate to config/wireguard/

# 1. Generate all required keys
./generate-keys.sh

# This creates:
# - Server private/public key pair
# - Peer keys for Desktop, VPS, iPhone, and Laptop
# - A .env file with all keys
# - Individual key files in keys/ directory
```

### Apply Configuration

```bash
# 2. Generate wg0.conf from the template
./apply-config.sh

# This reads the .env file and generates wg0.conf with all keys filled in
```

### Update Network Interface

**Important**: The default template uses `eth0` as the network interface. Update this if your system uses a different interface:

```bash
# Find your network interface name
ip link show

# Common interface names:
# - eth0 (common in VMs)
# - enp1s0 (PCI Ethernet)
# - eno1 (onboard Ethernet)
# - wlan0 (wireless)

# Edit the template or generated config to use your interface
sudo nano /etc/wireguard/wg0.conf

# Look for the PostUp/PostDown lines and update eth0 to your interface:
# PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -t nat -A POSTROUTING -o YOUR_INTERFACE -j MASQUERADE
```

### Export Peer Configurations

```bash
# 3. Generate client configuration files
./export-peer-configs.sh --endpoint your.public.ip:51820 \
    --allowed-ips 10.253.0.0/24 \
    --dns 1.1.1.1

# Replace:
# - your.public.ip: Your public IP or domain
# - allowed-ips: Networks accessible through the VPN
# - dns: DNS server for VPN clients

# This creates QR codes and config files in peer-configs/
```

### Deploy Configuration

```bash
# Copy the generated config to the system location
sudo cp wg0.conf /etc/wireguard/wg0.conf
sudo chmod 600 /etc/wireguard/wg0.conf

# Enable and start the WireGuard service
sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0

# Verify it's running
sudo wg show
```

### WireGuard Network Details

- **Server IP**: `10.253.0.1/24` (NAB9 mini PC)
- **Listen Port**: `51820`
- **Network Range**: `10.253.0.0/24`
- **Default Peers**:
  - LAN-Desktop: `10.253.0.6/32`
  - VPS: `10.253.0.8/32`
  - iPhone: `10.253.0.9/32`
  - Laptop: `10.253.0.11/32`

### Manual Configuration (Without Scripts)

If you prefer to configure manually:

```bash
# Generate server keys
wg genkey | sudo tee /etc/wireguard/server_private.key
sudo cat /etc/wireguard/server_private.key | wg pubkey | sudo tee /etc/wireguard/server_public.key

# Generate peer keys (repeat for each peer)
wg genkey | tee peer_private.key | wg pubkey > peer_public.key

# Create /etc/wireguard/wg0.conf manually using the template as reference
sudo nano /etc/wireguard/wg0.conf
```

---

## 4. NFS Mounts Setup

NFS mounts provide access to media and data stored on your file server.

### Create Mount Points

```bash
# Create directories where NFS shares will be mounted
sudo mkdir -p /mnt/nas-media
sudo mkdir -p /mnt/nas-nextcloud
sudo mkdir -p /mnt/nas-immich
sudo mkdir -p /mnt/nas-photos
```

### Configure NFS Mounts

The image includes systemd mount units in `/etc/systemd/system/`. You may need to update the NFS server IP.

#### Update Mount Units

If your NFS server IP differs from `192.168.7.10`, update the mount files:

```bash
# Find all mount units
ls /etc/systemd/system/*.mount

# Edit each one to update the NFS server IP
sudo nano /etc/systemd/system/mnt-nas-media.mount
```

Example mount unit (`/etc/systemd/system/mnt-nas-media.mount`):

```ini
[Unit]
Description=NFS mount for media storage (Plex/Jellyfin)
After=network-online.target
Wants=network-online.target
Before=docker.service

[Mount]
What=192.168.7.10:/mnt/storage/Media    # Update this IP
Where=/mnt/nas-media
Type=nfs
Options=ro,hard,intr,rsize=131072,wsize=131072,tcp,timeo=600,retrans=2,_netdev

TimeoutSec=60

[Install]
WantedBy=multi-user.target
WantedBy=remote-fs.target
```

#### Enable and Start Mounts

```bash
# Reload systemd to pick up changes
sudo systemctl daemon-reload

# Enable mounts to start at boot
sudo systemctl enable mnt-nas-media.mount
sudo systemctl enable mnt-nas-nextcloud.mount
sudo systemctl enable mnt-nas-immich.mount

# Start mounts now
sudo systemctl start mnt-nas-media.mount
sudo systemctl start mnt-nas-nextcloud.mount
sudo systemctl start mnt-nas-immich.mount

# Verify mounts
df -h | grep nas
mount | grep nfs
```

### Alternative: Using /etc/fstab

If you prefer traditional fstab entries instead of systemd mount units:

```bash
# Edit fstab
sudo nano /etc/fstab

# Add NFS mount entries
192.168.7.10:/mnt/storage/Media      /mnt/nas-media      nfs  defaults,ro,_netdev  0 0
192.168.7.10:/mnt/storage/Nextcloud  /mnt/nas-nextcloud  nfs  defaults,rw,_netdev  0 0
192.168.7.10:/mnt/storage/Photos     /mnt/nas-photos     nfs  defaults,ro,_netdev  0 0

# Mount all
sudo mount -a
```

**Note**: The `_netdev` option is crucial - it tells systemd to wait for the network before trying to mount.

### NFS Mount Options Explained

- `ro` / `rw`: Read-only or read-write
- `hard`: Retry indefinitely if server is unreachable (recommended for critical data)
- `soft`: Timeout and return error if server unreachable (alternative)
- `intr`: Allow interrupting NFS operations
- `rsize=131072,wsize=131072`: 128KB read/write buffer size (optimal for gigabit)
- `tcp`: Use TCP instead of UDP
- `timeo=600`: 60 second timeout (600 deciseconds)
- `retrans=2`: Number of retransmissions before failing
- `_netdev`: Wait for network before mounting

---

## 5. Container Setup

Choose between Podman Compose (recommended for CoreOS/immutable systems) or Docker.

### Option A: Podman Compose (Recommended)

uCore/Fedora CoreOS comes with Podman pre-installed. Podman is daemonless and more aligned with CoreOS philosophy.

#### Install Podman Compose

```bash
# Install podman-compose
pip3 install --user podman-compose

# OR use the system package if available
sudo rpm-ostree install podman-compose
sudo systemctl reboot
```

#### Create Compose Files

Navigate to your compose directory and create compose files for your services:

```bash
cd ~/compose/media
```

Create `docker-compose.yml` (or `compose.yaml`):

```yaml
version: "3.9"

services:
  plex:
    image: lscr.io/linuxserver/plex:latest
    container_name: plex
    network_mode: host
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=America/New_York
      - VERSION=docker
    volumes:
      - /home/dockeruser/appdata/plex:/config
      - /mnt/nas-media:/media:ro
    devices:
      - /dev/dri:/dev/dri  # Hardware transcoding
    restart: unless-stopped

  jellyfin:
    image: lscr.io/linuxserver/jellyfin:latest
    container_name: jellyfin
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=America/New_York
    volumes:
      - /home/dockeruser/appdata/jellyfin:/config
      - /mnt/nas-media:/media:ro
    ports:
      - 8096:8096
      - 8920:8920  # HTTPS
    devices:
      - /dev/dri:/dev/dri
    restart: unless-stopped

  overseerr:
    image: lscr.io/linuxserver/overseerr:latest
    container_name: overseerr
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=America/New_York
    volumes:
      - /home/dockeruser/appdata/overseerr:/config
    ports:
      - 5055:5055
    restart: unless-stopped
```

#### Create Environment File

```bash
# Create .env file for sensitive data
cat > ~/compose/media/.env << 'EOF'
# User/Group IDs (run `id` to find yours)
PUID=1000
PGID=1000

# Timezone
TZ=America/New_York

# Paths
APPDATA_DIR=/home/dockeruser/appdata
MEDIA_DIR=/mnt/nas-media

# Service-specific variables
PLEX_CLAIM=claim-xxxxxxxxxxxx
EOF
```

#### Start Services with Podman Compose

```bash
cd ~/compose/media
podman-compose up -d

# View logs
podman-compose logs -f

# Stop services
podman-compose down
```

#### Create Systemd Service for Auto-Start

```bash
# Create a user systemd service
mkdir -p ~/.config/systemd/user

cat > ~/.config/systemd/user/podman-compose-media.service << 'EOF'
[Unit]
Description=Podman Compose - Media Stack
Requires=network-online.target
After=network-online.target mnt-nas-media.mount

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/home/dockeruser/compose/media
ExecStart=/usr/local/bin/podman-compose up -d
ExecStop=/usr/local/bin/podman-compose down
TimeoutStartSec=0

[Install]
WantedBy=default.target
EOF

# Enable and start
systemctl --user enable podman-compose-media
systemctl --user start podman-compose-media

# Enable lingering so service starts at boot even when user not logged in
sudo loginctl enable-linger dockeruser
```

---

### Option B: Docker

If you prefer Docker over Podman:

#### Install Docker

```bash
# Layer Docker onto the CoreOS system
sudo rpm-ostree install docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Reboot to apply
sudo systemctl reboot

# After reboot, enable and start Docker
sudo systemctl enable docker
sudo systemctl start docker

# Add your user to docker group
sudo usermod -aG docker $USER

# Log out and back in for group changes to take effect
```

#### Create Docker Compose Files

The compose file format is identical to Podman:

```bash
cd ~/compose/media
nano docker-compose.yml
# (Use the same compose file content as shown in Podman section)
```

#### Start Services with Docker Compose

```bash
cd ~/compose/media
docker compose up -d

# View logs
docker compose logs -f

# Stop services
docker compose down
```

#### Create Systemd Service for Docker Compose

```bash
sudo nano /etc/systemd/system/docker-compose-media.service
```

```ini
[Unit]
Description=Docker Compose - Media Stack
Requires=docker.service network-online.target
After=docker.service network-online.target mnt-nas-media.mount

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/home/dockeruser/compose/media
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable docker-compose-media
sudo systemctl start docker-compose-media
```

---

## 6. Service Deployment

### Complete Stack Deployment

If you're using multiple compose files (recommended for organization):

```bash
# Deploy all stacks
cd ~/compose/media && podman-compose up -d
cd ~/compose/web && podman-compose up -d
cd ~/compose/cloud && podman-compose up -d
```

Or with a single consolidated compose file:

```bash
cd ~/compose && podman-compose up -d
```

### Verify Services

```bash
# Check running containers
podman ps
# OR
docker ps

# Check service health
curl http://localhost:8096  # Jellyfin
curl http://localhost:32400 # Plex
curl http://localhost:5055  # Overseerr
```

### Access Services

- **Plex**: `http://your-ip:32400/web`
- **Jellyfin**: `http://your-ip:8096`
- **Overseerr**: `http://your-ip:5055`
- **Nextcloud**: `http://your-ip:8080` (or via reverse proxy)
- **Immich**: `http://your-ip:2283` (or via reverse proxy)

---

## Troubleshooting

### NFS Mounts Not Working

```bash
# Check NFS server connectivity
ping 192.168.7.10

# Test manual mount
sudo mount -t nfs 192.168.7.10:/mnt/storage/Media /mnt/nas-media

# Check mount status
systemctl status mnt-nas-media.mount

# View detailed logs
journalctl -u mnt-nas-media.mount -f
```

### WireGuard Connection Issues

```bash
# Check WireGuard status
sudo wg show

# View WireGuard logs
journalctl -u wg-quick@wg0 -f

# Restart WireGuard
sudo systemctl restart wg-quick@wg0

# Test connectivity to peer
ping 10.253.0.8  # VPS
```

### Container Issues

```bash
# View container logs
podman logs <container-name>
docker logs <container-name>

# Restart a specific container
podman restart <container-name>
docker restart <container-name>

# Recreate containers
cd ~/compose/media
podman-compose down && podman-compose up -d
```

### Permission Issues

```bash
# Fix appdata permissions
sudo chown -R 1000:1000 ~/appdata

# Check SELinux context (if enabled)
ls -Z ~/appdata

# Fix SELinux labels if needed
sudo semanage fcontext -a -t container_file_t "~/appdata(/.*)?"
sudo restorecon -Rv ~/appdata
```

### System Updates Breaking Things

```bash
# Rollback to previous deployment
sudo rpm-ostree rollback
sudo systemctl reboot

# Pin current deployment to prevent updates
sudo rpm-ostree status
sudo ostree admin pin 0  # Pin index 0
```

---

## Next Steps

After completing this setup:

1. Configure each service through its web interface
2. Set up reverse proxy (Nginx Proxy Manager) on VPS if using WireGuard tunnel
3. Configure SSL certificates
4. Set up backups for appdata
5. Implement monitoring (optional)
6. Configure Fail2ban for additional security

For more information, see the main [README](../README.md).
