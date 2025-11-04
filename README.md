# homelab-coreos-minipc

This repository defines the **frontend application node** for my homelab — a complete, production-ready **uCore (Ublue CoreOS)** configuration for running all user-facing services on a NAB9 mini PC with Intel QuickSync hardware transcoding.

[![Build Status](https://github.com/yourusername/homelab-coreos-minipc/actions/workflows/build.yml/badge.svg)](https://github.com/yourusername/homelab-coreos-minipc/actions)

---

## Purpose

- 🎬 **Media Streaming** - Plex & Jellyfin with Intel QuickSync transcoding
- ☁️ **Personal Cloud** - Nextcloud AIO for files, calendar, contacts
- 📸 **Photo Backup** - Immich for automatic mobile photo backup
- 🎫 **Request Management** - Overseerr for media requests
- 🔐 **Secure Access** - WireGuard VPN tunnel to VPS for remote access
- 📦 **Immutable Infrastructure** - Declarative configuration with BlueBuild

---

## Quick Start

```bash
# 1. Clone this repository
git clone https://github.com/yourusername/homelab-coreos-minipc.git
cd homelab-coreos-minipc

# 2. Configure WireGuard
cd config/wireguard
cp wg0.conf.template wg0.conf
nano wg0.conf  # Fill in your keys

# 3. Configure Docker Compose
cd ../../compose
cp .env.example .env
nano .env  # Fill in your secrets

# 4. Run setup script
cd ..
sudo ./scripts/setup.sh

# 5. Start services
cd compose
docker compose -f media.yml -f web.yml -f cloud.yml up -d
```

📖 **Full setup guide:** [docs/SETUP.md](docs/SETUP.md)

---

## Stack Overview

| Component               | Role                                     |
|-------------------------|------------------------------------------|
| uCore (Ublue CoreOS)    | Immutable base OS on the NAB9 mini PC   |
| Docker / Podman Compose | Orchestrates all app containers         |
| WireGuard client        | Connects to DigitalOcean VPS            |
| Nginx Proxy Manager     | SSL / reverse proxy on VPS side         |
| UFW + Fail2ban          | Protects exposed media / SSH ports      |
| NFS client              | Mounts media from the file server       |
| Intel QuickSync         | Hardware video transcoding              |

---

## Network Architecture

```
Internet
   ├─→ Plex (32400) ─────────→ NAB9 Mini PC (Direct)
   ├─→ Jellyfin (8096/8920) ──→ NAB9 Mini PC (Direct)
   └─→ WireGuard Tunnel ──────→ VPS (DigitalOcean)
             ↓ (Nginx Proxy Manager)
       ┌─────┴─────────────────────┐
       ├─→ Overseerr               │
       ├─→ Nextcloud    10.99.0.2  │ NAB9 Mini PC
       ├─→ Immich                  │ (192.168.7.x)
       └─→ Wizarr                  │
             ↓
       NFS Mounts (Media, Data)
             ↓
       File Server (192.168.7.10)
```

---

## Repository Structure

```
homelab-coreos-minipc/
├── recipes/
│   └── recipe.yml              # Bluebuild recipe for uCore image
├── config/
│   ├── wireguard/
│   │   └── wg0.conf.template  # WireGuard VPN configuration
│   ├── nfs/
│   │   └── fstab.template     # NFS mount configuration
│   ├── gpu/
│   │   └── intel-qsv-setup.sh # Intel QuickSync setup script
│   └── firewall/
│       └── ufw-rules.sh       # Firewall configuration
├── compose/
│   ├── media.yml              # Plex, Jellyfin, Tautulli
│   ├── web.yml                # Overseerr, Wizarr, dashboards
│   ├── cloud.yml              # Nextcloud AIO, Immich
│   ├── .env.example           # Environment variables template
│   └── README.md              # Service documentation
├── scripts/
│   ├── setup.sh               # Initial system setup
│   ├── nfs-health.sh          # NFS mount health monitoring
│   ├── wireguard-check.sh     # VPN connection monitoring
│   └── gpu-verify.sh          # GPU transcoding verification
├── docs/
│   ├── SETUP.md               # Complete setup guide
│   ├── SERVICES.md            # Service configuration guide
│   ├── GPU_TRANSCODING.md     # GPU transcoding guide
│   └── TROUBLESHOOTING.md     # Common issues and solutions
├── files/
│   └── system/
│       └── etc/systemd/system/ # Systemd unit files
└── README.md
```

---

## Documentation

- 📘 [Setup Guide](docs/SETUP.md) - Complete installation and configuration
- 🛠️ [Services Guide](docs/SERVICES.md) - All services and their configuration
- 🎮 [GPU Transcoding](docs/GPU_TRANSCODING.md) - Intel QuickSync setup and optimization
- 🔧 [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues and solutions

---

## System Requirements

### Hardware
- Intel 12th gen+ CPU (with QuickSync iGPU)
- 16GB+ RAM
- 256GB+ NVMe SSD
- 2.5GbE ethernet

### Network
- Static IP on local network (192.168.7.x)
- NFS file server accessible (192.168.7.10)
- DigitalOcean VPS with WireGuard
- Port forwarding for Plex (32400) and Jellyfin (8096, 8920)

---

## Services

### 🎬 Media Services
- **Plex** - Media streaming with hardware transcoding
- **Jellyfin** - Open-source alternative to Plex
- **Tautulli** - Plex monitoring and statistics

### 🌐 Web Services
- **Overseerr** - Media request management
- **Wizarr** - Automated user invitations
- **Homepage** - Service dashboard (optional)

### ☁️ Cloud Services
- **Nextcloud AIO** - Personal cloud, files, calendar, contacts
- **Immich** - Self-hosted photo and video backup

### 🔒 Security & Networking
- **WireGuard VPN** - Secure tunnel to VPS for remote access
- **UFW Firewall** - Port-based access control
- **Fail2ban** - Intrusion prevention

### 🎮 Hardware Acceleration
- Intel QuickSync support for hardware transcoding
- 10-15x more efficient than CPU transcoding
- Support for multiple simultaneous 4K transcodes

---

## Updates & Rollbacks

### Update the OS

```bash
sudo rpm-ostree upgrade
sudo systemctl reboot
```

### Rebase to a New Image

```bash
rpm-ostree rebase ostree-unverified-registry:ghcr.io/yourusername/homelab-coreos-minipc:latest
```

### Rollback If Needed

```bash
rpm-ostree rollback
sudo systemctl reboot
```

---

## Security

- Only Plex and Jellyfin ports are forwarded from WAN to the mini PC
- All other applications are reached through the VPS (WireGuard + NPM)
- Fail2ban blocks repeated offenders on SSH and media ports
- Secrets (`.env`, WireGuard keys, Cloudflare tokens) are kept out of Git
- Firewall rules allow only what the stack actually uses

---

## Reboot Behavior

On boot:

1. WireGuard connects to the VPS
2. NFS mounts are brought up under `/mnt/nas-*`
3. The media stack starts via Docker Compose
4. Optional timers run health checks and log status

---

## Building the Image

This repository uses GitHub Actions to automatically build the custom uCore image.

**Manual build:**
```bash
# Install bluebuild
curl -L https://github.com/blue-build/cli/releases/latest/download/bluebuild-installer.sh | bash

# Build the image
bluebuild build recipes/recipe.yml

# Tag and push to registry
podman tag localhost/homelab-coreos-minipc:latest ghcr.io/yourusername/homelab-coreos-minipc:latest
podman push ghcr.io/yourusername/homelab-coreos-minipc:latest
```

**Rebase to custom image:**
```bash
sudo rpm-ostree rebase ostree-unverified-registry:ghcr.io/yourusername/homelab-coreos-minipc:latest
sudo systemctl reboot
```

---

## Philosophy

- **Declarative over click-ops:** The system is described in Git, not in wizards
- **Frontend vs backend separation:** This node runs the apps; the file server holds the data
- **Immutable host:** No ad-hoc package installs. Everything goes through the image
- **Fast rebuilds:** Reflash → rebase → reboot gets you back to the same state
- **Hardware optimization:** Intel QuickSync for efficient transcoding

---

## Contributing

This is a personal homelab configuration, but feel free to:
- Use this as a template for your own setup
- Report issues or suggest improvements
- Submit pull requests for bug fixes or enhancements

---

## License

MIT License — see `LICENSE` for details.

---

## Acknowledgments

- [Universal Blue](https://universal-blue.org/) - uCore base image
- [BlueBuild](https://blue-build.org/) - Declarative OS configuration
- [LinuxServer.io](https://www.linuxserver.io/) - Excellent container images
- The homelab community for inspiration and guidance
