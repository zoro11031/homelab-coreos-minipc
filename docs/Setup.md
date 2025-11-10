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
# Create main compose directory (system-wide location)
sudo mkdir -p /srv/containers

# Create subdirectories for different service stacks
sudo mkdir -p /srv/containers/media      # Media services (Plex, Jellyfin, etc.)
sudo mkdir -p /srv/containers/web        # Web services (Overseerr, Wizarr)
sudo mkdir -p /srv/containers/cloud      # Cloud services (Nextcloud, Immich)

# Set appropriate ownership (assuming dockeruser)
sudo chown -R dockeruser:dockeruser /srv/containers
```

### Create Application Data Directory

```bash
# Create appdata directory for persistent container data
sudo mkdir -p /var/lib/containers/appdata

# Create subdirectories for each service
sudo mkdir -p /var/lib/containers/appdata/plex
sudo mkdir -p /var/lib/containers/appdata/jellyfin
sudo mkdir -p /var/lib/containers/appdata/overseerr
sudo mkdir -p /var/lib/containers/appdata/wizarr
sudo mkdir -p /var/lib/containers/appdata/nextcloud
sudo mkdir -p /var/lib/containers/appdata/immich
sudo mkdir -p /var/lib/containers/appdata/postgres
sudo mkdir -p /var/lib/containers/appdata/redis

# Set appropriate ownership (assuming dockeruser)
sudo chown -R dockeruser:dockeruser /var/lib/containers/appdata
```

### Recommended Directory Structure

```
/srv/containers/            # Container orchestration files
├── media/
│   ├── compose.yml (or docker-compose.yml)
│   └── .env
├── web/
│   ├── compose.yml
│   └── .env
└── cloud/
    ├── compose.yml
    └── .env

/var/lib/containers/appdata/  # Application persistent data
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
/srv/containers/
├── compose.yml             # All services in one file
└── .env

/var/lib/containers/appdata/
└── [service directories as above]
```

---

## 3. WireGuard Configuration

WireGuard provides secure VPN connectivity for remote access and VPS tunneling.

### Generate WireGuard Keys and Configuration

The setup scripts are located in `~/setup/wireguard-setup/`:

```bash
# Navigate to the wireguard setup directory
cd ~/setup/wireguard-setup

# 1. Generate all required keys
./generate-keys.sh

# This creates:
# - keys/ directory with all key files
# - .env file with all keys and WG_OUTBOUND_INTERFACE
# - Server keys: server-private.key, server-public.key
# - Peer keys for: Desktop, VPS, iPhone, Laptop (private, public, and preshared keys)
```

The script will attempt to auto-detect your WAN interface and set `WG_OUTBOUND_INTERFACE` in the `.env` file. If detection fails or you need to change it, edit `.env` and update the value (e.g., `enp1s0`, `eno1`, etc.).

### Apply Configuration

```bash
# 2. Generate wg0.conf from the template
./apply-config.sh

# This script:
# - Reads keys from .env
# - Validates WG_OUTBOUND_INTERFACE (auto-detects if not set)
# - Substitutes placeholders in wg0.conf.template
# - Generates wg0.conf with proper permissions (600)
```

### Export Peer Configurations

```bash
# 3. Generate client configuration files for peers
./export-peer-configs.sh --endpoint your.public.ip:51820 \
    --allowed-ips 10.253.0.0/24 \
    --dns 1.1.1.1

# Options:
# --endpoint: Required. Your public IP or domain with port (e.g., vpn.example.com:51820)
# --allowed-ips: Networks accessible through VPN (default: 10.253.0.0/24)
# --dns: Optional DNS server for clients (e.g., 1.1.1.1)
# --output-dir: Directory for configs (default: ./peer-configs)

# This creates individual .conf files in peer-configs/:
# - desktop.conf
# - vps.conf
# - iphone.conf
# - laptop.conf
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
- **Configured Peers**:
  - LAN-Desktop-Justin: `10.253.0.6/32`
  - VPS: `10.253.0.8/32`
  - iPhone: `10.253.0.9/32`
  - Framework Laptop Justin: `10.253.0.11/32`

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

### Setup Templates

Compose templates and a `.env.example` are provided in `~/setup/compose-setup/`. Copy these to `/srv/containers/` as your starting point:

```bash
# Copy compose templates to working location
sudo cp -r ~/setup/compose-setup/* /srv/containers/
sudo chown -R core:core /srv/containers
cd /srv/containers
```

---

### Option A: Podman Compose (Recommended)

Podman Compose is pre-installed in the image.

#### Setup Compose Files

Templates are provided in `~/setup/compose-setup/`:
- `media.yml` - Plex, Jellyfin, Tautulli
- `web.yml` - Overseerr, Wizarr, Organizr, Homepage
- `cloud.yml` - Nextcloud (with PostgreSQL, Redis, Collabora) and Immich
- `.env.example` - Environment variables template

```bash
# Copy templates to /srv/containers
sudo cp ~/setup/compose-setup/*.yml /srv/containers/
sudo cp ~/setup/compose-setup/.env.example /srv/containers/.env

# Set ownership
sudo chown -R dockeruser:dockeruser /srv/containers

# Edit the .env file with your actual values
cd /srv/containers
nano .env
```

#### Configure Environment Variables

Edit `/srv/containers/.env` and configure:

**Required:**
- `PUID` / `PGID` - Your user/group IDs (run `id` to find)
- `TZ` - Your timezone (e.g., `America/New_York`)
- `APPDATA_PATH` - Path to appdata (default: `/var/lib/containers/appdata`)

**Service-specific:**
- `PLEX_CLAIM_TOKEN` - Get from https://plex.tv/claim
- `NEXTCLOUD_DB_PASSWORD` - Database password
- `NEXTCLOUD_TRUSTED_DOMAINS` - Your Nextcloud domain
- `IMMICH_DB_PASSWORD` - Immich database password
- `COLLABORA_PASSWORD` - Collabora admin password

See `.env.example` for all available options.

#### Start Services with Podman Compose

```bash
# Start individual stacks
cd /srv/containers
podman-compose -f media.yml up -d
podman-compose -f web.yml up -d
podman-compose -f cloud.yml up -d

# View logs
podman-compose -f media.yml logs -f

# Stop services
podman-compose -f media.yml down
```

# Stop services
podman-compose down
```

#### Create Systemd Services for Auto-Start

Create separate services for each stack:

```bash
# Media stack service
sudo nano /etc/systemd/system/podman-compose-media.service
```

```ini
[Unit]
Description=Podman Compose - Media Stack
Requires=network-online.target
After=network-online.target mnt-nas-media.mount

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/srv/containers
ExecStart=/usr/local/bin/podman-compose -f media.yml up -d
ExecStop=/usr/local/bin/podman-compose -f media.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

```bash
# Web stack service
sudo nano /etc/systemd/system/podman-compose-web.service
```

```ini
[Unit]
Description=Podman Compose - Web Stack
Requires=network-online.target
After=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/srv/containers
ExecStart=/usr/local/bin/podman-compose -f web.yml up -d
ExecStop=/usr/local/bin/podman-compose -f web.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

```bash
# Cloud stack service
sudo nano /etc/systemd/system/podman-compose-cloud.service
```

```ini
[Unit]
Description=Podman Compose - Cloud Stack
Requires=network-online.target
After=network-online.target mnt-nas-nextcloud.mount mnt-nas-immich.mount

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/srv/containers
ExecStart=/usr/local/bin/podman-compose -f cloud.yml up -d
ExecStop=/usr/local/bin/podman-compose -f cloud.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start services
sudo systemctl daemon-reload
sudo systemctl enable podman-compose-media
sudo systemctl enable podman-compose-web
sudo systemctl enable podman-compose-cloud

sudo systemctl start podman-compose-media
sudo systemctl start podman-compose-web
sudo systemctl start podman-compose-cloud
```

---

### Option B: Docker

Docker and Docker Compose are pre-installed in the image.

#### Setup Compose Files

Use the same templates from `~/setup/compose-setup/`:

```bash
# Copy templates to /srv/containers
sudo cp ~/setup/compose-setup/*.yml /srv/containers/
sudo cp ~/setup/compose-setup/.env.example /srv/containers/.env

# Set ownership
sudo chown -R dockeruser:dockeruser /srv/containers

# Edit the .env file
cd /srv/containers
nano .env
```

#### Start Services with Docker Compose

```bash
cd /srv/containers
docker compose -f media.yml up -d
docker compose -f web.yml up -d
docker compose -f cloud.yml up -d

# View logs
docker compose -f media.yml logs -f

# Stop services
docker compose -f media.yml down
```

#### Create Systemd Services for Docker Compose

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
WorkingDirectory=/srv/containers
ExecStart=/usr/bin/docker compose -f media.yml up -d
ExecStop=/usr/bin/docker compose -f media.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

Create similar services for `web.yml` and `cloud.yml`, then enable:

```bash
# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable docker-compose-media
sudo systemctl enable docker-compose-web
sudo systemctl enable docker-compose-cloud

sudo systemctl start docker-compose-media
sudo systemctl start docker-compose-web
sudo systemctl start docker-compose-cloud
```

---

## 6. Service Deployment

### Complete Stack Deployment

Deploy all stacks using the provided compose files:

```bash
cd /srv/containers

# Start all services
podman-compose -f media.yml up -d
podman-compose -f web.yml up -d
podman-compose -f cloud.yml up -d
```

Or if using systemd services:

```bash
sudo systemctl start podman-compose-media
sudo systemctl start podman-compose-web
sudo systemctl start podman-compose-cloud
```

### Verify Services

```bash
# Check running containers
podman ps
# OR
docker ps

# Check service health
curl http://localhost:8096  # Jellyfin
curl http://localhost:32400 # Plex (if network_mode: host)
curl http://localhost:5055  # Overseerr
curl http://localhost:8080  # Nextcloud
curl http://localhost:2283  # Immich
```

### Access Services

**Media Services (Direct WAN Access):**
- **Plex**: `http://your-ip:32400/web`
- **Jellyfin**: `http://your-ip:8096`

**Web Services (via VPS Proxy):**
- **Overseerr**: `http://your-ip:5055`
- **Wizarr**: `http://your-ip:5690`
- **Organizr**: `http://your-ip:9983`
- **Homepage**: `http://your-ip:3000`

**Cloud Services (via VPS Proxy):**
- **Nextcloud**: `http://your-ip:8080`
- **Collabora**: `http://your-ip:9980`
- **Immich**: `http://your-ip:2283`

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
cd /srv/containers
podman-compose -f media.yml down && podman-compose -f media.yml up -d
```

### Permission Issues

```bash
# Fix appdata permissions
sudo chown -R 1000:1000 /var/lib/containers/appdata

# Check SELinux context (if enabled)
ls -Z /var/lib/containers/appdata

# Fix SELinux labels if needed
sudo semanage fcontext -a -t container_file_t "/var/lib/containers/appdata(/.*)?"
sudo restorecon -Rv /var/lib/containers/appdata
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
