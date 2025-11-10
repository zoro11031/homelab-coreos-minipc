# UBlue uCore Homelab Setup Scripts

Comprehensive interactive bash setup scripts for configuring a homelab environment on **UBlue uCore** (immutable Fedora with rpm-ostree).

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [System Requirements](#system-requirements)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Script Descriptions](#script-descriptions)
- [Directory Structure](#directory-structure)
- [Configuration](#configuration)
- [Usage](#usage)
- [Troubleshooting](#troubleshooting)
- [Advanced Topics](#advanced-topics)
- [FAQ](#faq)

## Overview

These scripts automate the complete setup of a homelab environment on **UBlue uCore**, an immutable Fedora-based operating system that uses rpm-ostree for package management. The setup includes:

- **Media Services**: Plex, Jellyfin, Tautulli
- **Web Services**: Overseerr, Wizarr, Organizr, Homepage
- **Cloud Services**: Nextcloud, Immich, Collabora
- **Infrastructure**: WireGuard VPN, NFS network storage, systemd services

### What Makes This Special?

- **Immutable OS Support**: Designed specifically for UBlue uCore's read-only filesystem
- **BlueBuild Integration**: Works with pre-configured systemd services baked into custom images
- **Template-Based**: Uses compose templates from first-boot home directory setup
- **Interactive Configuration**: Guided setup with sensible defaults
- **Idempotent**: Safe to re-run without breaking existing configuration
- **Comprehensive Diagnostics**: Built-in troubleshooting tools

## Features

✅ **Pre-flight System Verification**
- Checks rpm-ostree environment
- Detects pre-existing services from BlueBuild image
- Validates required packages and templates

✅ **User Account Management**
- Creates dedicated container user (optional)
- Configures groups and permissions
- Sets up subuid/subgid for rootless containers

✅ **Directory Structure Creation**
- `/srv/containers/` for compose files
- `/var/lib/containers/appdata/` for persistent data
- NFS mount points

✅ **WireGuard VPN Configuration**
- Automatic key generation
- WAN interface auto-detection
- Peer configuration export

✅ **NFS Network Storage**
- Systemd mount unit creation
- Automatic server detection
- Mount verification

✅ **Container Service Setup**
- Template copying and customization
- Environment file generation
- Interactive password/token configuration

✅ **Service Deployment**
- Systemd service enablement
- Container image pulling
- Health checks and verification

✅ **Troubleshooting Tools**
- System diagnostics
- Service status checks
- Log collection

## System Requirements

### Operating System
- **UBlue uCore** (Fedora-based immutable OS)
- rpm-ostree package management
- systemd

### Required Packages
These should be layered into your BlueBuild image or installed via rpm-ostree:

```bash
- podman
- podman-compose
- nfs-utils
- wireguard-tools (optional)
```

### Hardware Requirements
- **Minimum Disk Space**:
  - 20GB on root filesystem
  - 50GB on /var filesystem
- **RAM**: 8GB minimum (16GB+ recommended)
- **CPU**: 2+ cores recommended
- **Network**: NFS server access (if using network storage)

### Prerequisites
1. UBlue uCore installed and running
2. User account with sudo access
3. Internet connectivity
4. (Optional) NFS server configured with exports
5. (Optional) BlueBuild custom image with pre-configured services

## Quick Start

### 1. Download Scripts

```bash
# Clone or extract the homelab-setup-scripts directory
cd ~
# Assuming scripts are in ~/homelab-setup-scripts
```

### 2. Run Setup

```bash
cd homelab-setup-scripts
./homelab-setup.sh
```

### 3. Choose Setup Mode

**Option A: Complete Setup**
- Runs all steps including WireGuard
- Takes 15-30 minutes

**Option Q: Quick Setup**
- Skips WireGuard configuration
- Takes 10-20 minutes

**Individual Steps**
- Run specific steps as needed

### 4. Access Services

After deployment, access your services at:

```
http://<your-ip>:32400/web  # Plex
http://<your-ip>:8096       # Jellyfin
http://<your-ip>:5055       # Overseerr
http://<your-ip>:8080       # Nextcloud
http://<your-ip>:2283       # Immich
# ... and more
```

## Architecture

### System Layout

```
/
├── etc/
│   ├── systemd/system/          # User-modified service units
│   └── wireguard/               # WireGuard configuration
├── srv/
│   └── containers/              # Compose files
│       ├── media/
│       │   ├── compose.yml
│       │   └── .env
│       ├── web/
│       │   ├── compose.yml
│       │   └── .env
│       └── cloud/
│           ├── compose.yml
│           └── .env
├── var/
│   └── lib/containers/
│       └── appdata/             # Persistent container data
│           ├── plex/
│           ├── jellyfin/
│           ├── nextcloud/
│           └── ...
├── mnt/
│   ├── nas-media/               # NFS mounts
│   ├── nas-nextcloud/
│   └── nas-immich/
└── usr/
    ├── lib/systemd/system/      # Pre-configured services (read-only)
    └── share/
        ├── compose-setup/       # Template compose files
        └── wireguard-setup/     # Template WireGuard configs
```

### Service Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Systemd Services                      │
│  ┌──────────────────────────────────────────────────┐  │
│  │  podman-compose-media.service                     │  │
│  │  podman-compose-web.service                       │  │
│  │  podman-compose-cloud.service                     │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                  Podman Containers                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │    Plex     │  │  Jellyfin   │  │  Nextcloud  │    │
│  │  Tautulli   │  │  Overseerr  │  │   Immich    │    │
│  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                   Storage Layer                          │
│  ┌──────────────┐        ┌──────────────┐              │
│  │ Local AppData│        │  NFS Mounts  │              │
│  │ /var/lib/... │        │  /mnt/nas-*  │              │
│  └──────────────┘        └──────────────┘              │
└─────────────────────────────────────────────────────────┘
```

## Script Descriptions

### Main Orchestrator

**`homelab-setup.sh`**
- Interactive menu-driven interface
- Run all steps or individual scripts
- Progress tracking
- Built-in help system

### Setup Scripts

**`00-preflight-check.sh`**
- Verifies UBlue uCore environment
- Checks required packages
- Detects pre-existing systemd services
- Validates template locations
- Tests network connectivity
- Checks disk space

**`01-user-setup.sh`**
- Creates/configures user account
- Adds to necessary groups (wheel, podman)
- Configures sudo access
- Sets up subuid/subgid mappings
- Detects UID/GID and timezone

**`02-directory-setup.sh`**
- Creates `/srv/containers/{media,web,cloud}/`
- Creates `/var/lib/containers/appdata/{service}/`
- Creates NFS mount point directories
- Sets proper ownership and permissions

**`03-wireguard-setup.sh`**
- Locates WireGuard templates
- Generates server and peer keys
- Auto-detects WAN interface
- Creates wg0.conf configuration
- Exports peer configurations
- Configures firewall rules

**`04-nfs-setup.sh`**
- Detects pre-existing systemd mount units
- Tests NFS server connectivity
- Creates/updates systemd mount units
- Enables and starts mounts
- Verifies mount status

**`05-container-setup.sh`**
- Copies compose templates to `/srv/containers/`
- Creates `.env` files for each stack
- Configures environment variables:
  - PUID/PGID (auto-detected)
  - Timezone (auto-detected)
  - Service passwords and tokens
- Sets proper ownership

**`06-service-deployment.sh`**
- Detects pre-configured systemd services
- Creates service units if needed
- Pulls container images
- Enables and starts services
- Verifies container health
- Displays access URLs

### Utility Scripts

**`troubleshoot.sh`**
- System information display
- Service status checks
- Container diagnostics
- Network connectivity tests
- NFS mount verification
- Disk usage analysis
- Log collection
- Common issues and solutions

**`common-functions.sh`**
- Shared functions library
- Color-coded output
- Configuration management
- Input validation
- Error handling
- Progress indicators

## Directory Structure

### Container Services

```
/srv/containers/
├── media/
│   ├── compose.yml          # Plex, Jellyfin, Tautulli
│   └── .env                 # Media stack environment
├── web/
│   ├── compose.yml          # Overseerr, Wizarr, Organizr, Homepage
│   └── .env                 # Web stack environment
└── cloud/
    ├── compose.yml          # Nextcloud, Immich, Collabora
    └── .env                 # Cloud stack environment
```

### Application Data

```
/var/lib/containers/appdata/
├── plex/                    # Plex configuration and database
├── jellyfin/                # Jellyfin configuration and database
├── tautulli/                # Tautulli statistics database
├── overseerr/               # Overseerr configuration
├── wizarr/                  # Wizarr invitation system
├── organizr/                # Organizr dashboard
├── homepage/                # Homepage dashboard
├── nextcloud/               # Nextcloud data and config
├── nextcloud-db/            # PostgreSQL for Nextcloud
├── nextcloud-redis/         # Redis for Nextcloud
├── collabora/               # Collabora Office
├── immich/                  # Immich photo management
├── immich-db/               # PostgreSQL for Immich
├── immich-redis/            # Redis for Immich
└── immich-ml/               # Immich machine learning
```

## Configuration

### Configuration File

All settings are saved in `~/.homelab-setup.conf`:

```bash
SETUP_USER=core
PUID=1000
PGID=1000
TZ=America/Chicago
APPDATA_PATH=/var/lib/containers/appdata
NFS_SERVER=192.168.7.10
WG_SERVER_IP=10.253.0.1/24
# ... and more
```

### Setup Markers

Completion status is tracked in `~/.local/homelab-setup/`:

```
preflight-complete
user-setup-complete
directory-setup-complete
wireguard-setup-complete
nfs-setup-complete
container-setup-complete
service-deployment-complete
```

### Environment Variables

Each service stack has its own `.env` file with:

**Common Variables:**
```bash
PUID=1000                    # User ID
PGID=1000                    # Group ID
TZ=America/Chicago           # Timezone
APPDATA_PATH=/var/lib/containers/appdata
```

**Media Stack:**
```bash
PLEX_CLAIM_TOKEN=claim-xxx
JELLYFIN_PUBLIC_URL=https://jellyfin.example.com
```

**Cloud Stack:**
```bash
NEXTCLOUD_ADMIN_USER=admin
NEXTCLOUD_ADMIN_PASSWORD=xxx
NEXTCLOUD_DB_PASSWORD=xxx
NEXTCLOUD_TRUSTED_DOMAINS=cloud.example.com
IMMICH_DB_PASSWORD=xxx
COLLABORA_PASSWORD=xxx
```

## Usage

### Running Complete Setup

```bash
./homelab-setup.sh
# Select: [A] Run All Steps
```

### Running Individual Steps

```bash
# Run pre-flight check only
./scripts/00-preflight-check.sh

# Configure NFS only
./scripts/04-nfs-setup.sh

# Deploy services only
./scripts/06-service-deployment.sh
```

### Reconfiguring

Scripts are idempotent and will detect existing configuration:

```bash
# Re-run user setup
./scripts/01-user-setup.sh
# Script will ask: "Reconfigure user setup? [y/N]"
```

### Resetting Setup

```bash
./homelab-setup.sh
# Select: [R] Reset Setup
# This removes markers and backs up configuration
```

## Troubleshooting

### Running Diagnostics

```bash
# Interactive troubleshooting menu
./scripts/troubleshoot.sh

# Run all diagnostics
./scripts/troubleshoot.sh --all

# Check services only
./scripts/troubleshoot.sh --services

# Check network only
./scripts/troubleshoot.sh --network

# Collect diagnostic logs
./scripts/troubleshoot.sh --logs
```

### Common Issues

#### Services Not Starting

```bash
# Check service status
sudo systemctl status podman-compose-media.service

# View service logs
sudo journalctl -u podman-compose-media.service -n 50

# Restart service
sudo systemctl restart podman-compose-media.service
```

#### NFS Mounts Failing

```bash
# Test NFS server connectivity
ping <nfs-server-ip>

# Check available exports
showmount -e <nfs-server-ip>

# View mount logs
sudo journalctl -u mnt-nas-media.mount

# Try manual mount
sudo mount -t nfs <server>:<export> /mnt/nas-media
```

#### Containers Not Running

```bash
# List all containers
podman ps -a

# View container logs
podman logs <container-name>

# Restart containers
cd /srv/containers/media
podman-compose down
podman-compose up -d
```

#### Permission Errors

```bash
# Check ownership
ls -la /srv/containers
ls -la /var/lib/containers/appdata

# Fix ownership
sudo chown -R <user>:<user> /srv/containers
sudo chown -R <user>:<user> /var/lib/containers/appdata
```

### Service Management

```bash
# View service status
sudo systemctl status podman-compose-*.service

# Start service
sudo systemctl start podman-compose-media.service

# Stop service
sudo systemctl stop podman-compose-media.service

# Restart service
sudo systemctl restart podman-compose-media.service

# View logs
sudo journalctl -u podman-compose-media.service -f
```

### Container Management

```bash
# List running containers
podman ps

# View container logs
podman logs <container-name>

# Follow container logs
podman logs -f <container-name>

# Enter container shell
podman exec -it <container-name> /bin/bash

# Restart container
podman restart <container-name>
```

### Using podman-compose

```bash
cd /srv/containers/<service>

# List containers
podman-compose ps

# View logs
podman-compose logs

# Follow logs
podman-compose logs -f

# Stop containers
podman-compose down

# Start containers
podman-compose up -d

# Pull latest images
podman-compose pull

# Restart containers
podman-compose restart
```

## Advanced Topics

### Working with Immutable OS

#### Installing Additional Packages

```bash
# Layer packages with rpm-ostree
sudo rpm-ostree install <package-name>

# Reboot to apply changes
sudo systemctl reboot

# Check deployment status
rpm-ostree status

# Rollback if needed
rpm-ostree rollback
sudo systemctl reboot
```

#### Modifying System Services

```bash
# System services in /usr/lib are read-only
# Copy to /etc/systemd/system to modify

sudo cp /usr/lib/systemd/system/example.service /etc/systemd/system/
sudo nano /etc/systemd/system/example.service
sudo systemctl daemon-reload
```

### Customizing Services

#### Modifying Compose Files

```bash
# Edit compose file
nano /srv/containers/media/compose.yml

# Restart service to apply changes
sudo systemctl restart podman-compose-media.service
```

#### Changing Environment Variables

```bash
# Edit environment file
nano /srv/containers/media/.env

# Restart service to apply changes
sudo systemctl restart podman-compose-media.service
```

### Backup and Restore

#### Backing Up Configuration

```bash
# Backup compose files
tar -czf ~/homelab-compose-backup.tar.gz /srv/containers

# Backup configuration
cp ~/.homelab-setup.conf ~/homelab-setup.conf.backup

# Backup application data (careful - can be large!)
tar -czf ~/homelab-appdata-backup.tar.gz /var/lib/containers/appdata
```

#### Restoring Configuration

```bash
# Restore compose files
sudo tar -xzf ~/homelab-compose-backup.tar.gz -C /

# Restore configuration
cp ~/homelab-setup.conf.backup ~/.homelab-setup.conf

# Restart services
sudo systemctl restart podman-compose-*.service
```

### Updating Container Images

```bash
# Pull latest images for a service
cd /srv/containers/media
podman-compose pull

# Restart service with new images
sudo systemctl restart podman-compose-media.service

# Or update all services
for service in media web cloud; do
    cd /srv/containers/$service
    podman-compose pull
done

sudo systemctl restart podman-compose-*.service
```

## FAQ

**Q: Can I run this on regular Fedora or other distributions?**
A: These scripts are designed for UBlue uCore's immutable filesystem. They may work on regular Fedora with modifications, but are not tested for that use case.

**Q: Do I need to use the BlueBuild custom image?**
A: No, but it's recommended. The scripts will create necessary services if they don't exist in your image.

**Q: Can I skip WireGuard setup?**
A: Yes, use the "Quick Setup" option or run individual steps, skipping step 3.

**Q: How do I add more services?**
A: Create a new compose file in `/srv/containers/<name>/`, create a `.env` file, and either create a systemd service or use `podman-compose` directly.

**Q: Can I use Docker instead of Podman?**
A: No, UBlue uCore uses Podman. However, compose files are compatible.

**Q: How do I update the system?**
A: Use `rpm-ostree upgrade` to update the system, then reboot.

**Q: Where are container logs stored?**
A: Podman logs are available via `podman logs <container>` and journald via `sudo journalctl -u <service>`.

**Q: Can I run services as root?**
A: Yes, but rootless containers (running as regular user) are recommended for security.

**Q: How do I completely remove everything?**
A: Run `sudo systemctl stop podman-compose-*.service`, delete directories in `/srv/containers` and `/var/lib/containers/appdata`, and remove configuration files.

## Contributing

Contributions are welcome! Please:
1. Test thoroughly on UBlue uCore
2. Follow existing code style
3. Update documentation
4. Add comments for complex logic

## License

MIT License - see LICENSE file for details

## Acknowledgments

- UBlue Project for the immutable Fedora base
- BlueBuild for custom image building
- All the open-source projects that make this homelab possible

## Support

For issues and questions:
- Check troubleshooting section
- Run diagnostics: `./scripts/troubleshoot.sh`
- Review logs: `sudo journalctl -u <service>`
- Check container status: `podman ps -a`

---

**Happy Homelabbing! 🚀**
