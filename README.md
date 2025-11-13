# homelab-coreos-minipc &nbsp; [![build](https://github.com/zoro11031/homelab-coreos-minipc/actions/workflows/build.yml/badge.svg)](https://github.com/zoro11031/homelab-coreos-minipc/actions/workflows/build.yml)

**Frontend application node** for my homelab — a declarative **uCore (Ublue CoreOS)** build running all user-facing services on a NAB9 mini PC.  
Pulls media from the file server over NFS and exposes select services to the internet through a WireGuard-linked VPS.

---

## Purpose

- Host Plex, Jellyfin, Nextcloud, Immich, Overseerr, Wizarr, etc.  
- Handle transcoding and external traffic.  
- Stay reproducible via uCore + BlueBuild.  
- Serve as the only WAN-exposed node.

---

## Stack Overview

| Component            | Role                                         |
|----------------------|----------------------------------------------|
| **uCore (Ublue CoreOS)** | Immutable host OS on NAB9 mini PC          |
| **Podman Compose**   | Container orchestration                      |
| **WireGuard**        | Encrypted tunnel to DigitalOcean VPS         |
| **Nginx Proxy Manager** | Reverse proxy + SSL termination on VPS     |
| **Fail2ban / Firewall** | Hardens SSH and media endpoints           |
| **NFS Client**       | Mounts media from the backend file server    |

---

## Network Architecture
```
                     Internet
                        │
    ┌───────────────────┴─────────────────────┐
    │                                         │
    ↓                                         ↓
┌────────────────────────┐          Direct Ports 8096/8920 & 32400
│   DigitalOcean VPS     │          (NAB9 Mini PC only)
│  ┌──────────────────┐  │                    │
│  │ Nginx Proxy      │  │                     │
│  │ Manager (SSL)    │  │                     │
│  │ Routes: Overseerr│  │                     │
│  │ Wizarr / Immich  │  │                     │
│  │ Nextcloud        │  │                     │
│  └──────────────────┘│                     │
│                        │                     │
│   WireGuard Tunnel     │                     │
│    (encrypted)         │                     │
└────────────┬───────────┘                   │
             │                                 │
             ↓                                 │
┌────────────────────────────────────────────┴─────────────┐
│              NAB9 Mini PC (uCore Frontend)               │
│  • Podman stack: Jellyfin, Plex, Overseerr, Wizarr, etc. │
│  • Direct exposure for Jellyfin/Plex                     │
│  • WAN ingress via VPS                                   │
└───────────────┬──────────────────────────────────────────┘
                  │
         LAN (NFS Access)
                 │
                 ↓
┌───────────────────────────────────────────┐
│      File Server (Ublue CoreOS)               │
│  • SnapRAID + mergerfs / NFS / qBittorrentVPN |
│  • Sonarr / Radarr / Lidarr / Prowlarr        │
│  • LAN-only access                            │
└───────────────────────────────────────────┘
                │
                ↓
          DAS Storage
```

---

## Declarative Build

**Base Image:** [uCore (Ublue CoreOS)](https://github.com/ublue-os/ucore)  
**Custom Image:** `ghcr.io/zoro11031/homelab-coreos-minipc:latest`

Built with **BlueBuild** to include:

- `wireguard-tools`, `podman`, `nfs-utils`, `fail2ban`, `zsh`, Intel VAAPI drivers  
- Predefined systemd units for NFS mounts, WireGuard, and container startup  
- Template for `/etc/wireguard/wg0.conf`  
- Empty `.dotfiles/` directory under `core` user for local customization  

---

## Core Services

| Service | Function | Access |
|----------|-----------|--------|
| **Plex** | Media server, Intel QuickSync HW transcoding | Direct port `32400` |
| **Jellyfin** | Open-source media server | Direct ports `8096/8920` |
| **Overseerr** | Media request manager | via VPS proxy |
| **Wizarr** | Invite automation | via VPS proxy |
| **Nextcloud** | Cloud + Collabora + Redis + PostgreSQL | via VPS proxy |
| **Immich** | Photo/video backup | via VPS proxy |

All containers are defined in `compose/minipc.yml`.

---

## Networking

### WireGuard

The mini PC runs as a WireGuard server.

- **Server:** `10.253.0.1/24`  
- **Port:** `51820`  
- **Peers:** VPS, personal devices, LAN nodes  
- Template: `config/wireguard/wg0.conf.template`  
- Service: `wg-quick@wg0.service` (autostart)

### NFS Mounts

Configured under `/etc/fstab` for media access, e.g.:
```
192.168.7.10:/mnt/storage/Media      /mnt/nas-media      nfs  defaults,ro  0 0
192.168.7.10:/mnt/storage/Nextcloud  /mnt/nas-nextcloud  nfs  defaults,ro  0 0
192.168.7.10:/mnt/storage/Photos     /mnt/nas-photos     nfs  defaults,ro  0 0
```

---

## Installation

### Option A — Fedora CoreOS ISO (Recommended)

1. Generate Ignition file using the `ignition/config.bu.template` and helper scripts.  
2. Download Fedora CoreOS ISO:  
```bash
   podman run --rm -v "$(pwd)":/data quay.io/coreos/coreos-installer:release download -s stable -p metal -f iso
```
3. Flash ISO to USB (`dd` or Rufus).
4. Boot target machine and run:
```bash
   sudo coreos-installer install /dev/sda --ignition-file /path/to/config.ign
```
5. Reboot. System will provision, then automatically rebase into `homelab-coreos-minipc` image.

### Option B — Prebuilt Custom ISO

Build with BlueBuild:
```bash
sudo bluebuild generate-iso --iso-name homelab-coreos-minipc.iso \
  image ghcr.io/zoro11031/homelab-coreos-minipc
```

---

## Post-Install Setup

After first boot, the system automatically copies setup scripts to `~/setup/`:
- `~/setup/home-lab-setup-scripts/` - Interactive homelab setup system
- `~/setup/compose-setup/` - Docker compose templates
- `~/setup/wireguard-setup/` - WireGuard configuration helpers

### Automated Setup (Recommended)

**Option 1: Go CLI (New, Recommended)**

The system includes a compiled Go binary for homelab setup:

```bash
# Interactive menu (default)
homelab-setup

# Or run specific commands
homelab-setup run all              # Run all setup steps
homelab-setup run quick            # Skip WireGuard
homelab-setup run preflight        # Run individual step
homelab-setup status               # Show setup status
homelab-setup reset                # Reset progress markers
homelab-setup --help               # Show all commands
```

**Available Commands:**
- `homelab-setup` - Launch interactive menu
- `homelab-setup run [all|quick|step]` - Run setup steps
- `homelab-setup status` - Show completion status
- `homelab-setup reset` - Clear all completion markers
- `homelab-setup troubleshoot` - Run diagnostics
- `homelab-setup version` - Show version info

**Setup Steps:**
1. Pre-flight Check - Verify system requirements
2. User Setup - Configure user account and permissions
3. Directory Setup - Create directory structure
4. WireGuard Setup - Configure VPN (optional)
5. NFS Setup - Configure network storage
6. Container Setup - Configure container services
7. Service Deployment - Deploy and start services

**Non-Interactive Mode:**

For automation or scripts:
```bash
homelab-setup run all \
  --non-interactive \
  --setup-user=containeruser \
  --nfs-server=192.168.7.10 \
  --homelab-base-dir=/mnt/homelab \
  --skip-wireguard
```

**Option 2: Bash Script (Legacy)**

The original bash script is also available:
```bash
cd ~/setup/home-lab-setup-scripts
./homelab-setup.sh
```

### Manual Setup

Alternatively, configure manually:
1. Configure WireGuard peers and keys.
2. Mount NFS shares for media and appdata.
3. Place compose files under `/srv/containers/`.
4. Start stack via systemd unit or Podman Compose.
5. Confirm WAN routing via VPS (reverse proxy).

Detailed instructions live in the [Setup Wiki](../../wiki/Setup).

---

## Updates & Rollbacks
```bash
sudo rpm-ostree upgrade && sudo systemctl reboot     # Update OS
sudo rpm-ostree rollback                             # Revert last deployment
```

---

## Boot Sequence

1. WireGuard link established
2. NFS mounts activated
3. Compose stack launched
4. Health timers run checks and log status

---

**Base:** `ghcr.io/ublue-os/ucore:latest`  
**Extensions:** WireGuard, NFS, zsh, fail2ban, Intel media stack
