# homelab-coreos-minipc &nbsp; [![build](https://github.com/zoro11031/homelab-coreos-minipc/actions/workflows/build.yml/badge.svg)](https://github.com/zoro11031/homelab-coreos-minipc/actions/workflows/build.yml)

This repository defines the **frontend application node** for my homelab — a declarative **uCore (Ublue CoreOS)** build that runs all user-facing services on a NAB9 mini PC.
It pulls media from the file server over NFS and exposes services to the internet via direct ports and a WireGuard-linked VPS.

---

## Purpose

- Run Plex, Jellyfin, Nextcloud, Immich, Overseerr, Wizarr, etc.  
- Handle all transcoding and outbound traffic to users.  
- Keep state declarative and rebuildable with uCore + BlueBuild.  
- Be the only node that has WAN ingress.

---

## Stack Overview

| Component               | Role                                     |
|-------------------------|------------------------------------------|
| uCore (Ublue CoreOS)    | Immutable base OS on the NAB9 mini PC   |
| Docker / Podman Compose | Orchestrates all app containers         |
| WireGuard client        | Connects to DigitalOcean VPS            |
| Nginx Proxy Manager     | SSL / reverse proxy on VPS side         |
| Fail2ban + firewall     | Protects exposed media / SSH ports      |
| NFS client              | Mounts media from the file server       |

---

## Architecture

    Internet
       ├─→ Plex (32400) — direct WAN to NAB9
       ├─→ Jellyfin (8096/8920) — direct WAN to NAB9
       └─→ WireGuard Tunnel → VPS (NPM)
                ↓
          Overseerr / Wizarr / Nextcloud / Immich
                ↓
          NFS over LAN (192.168.7.x)
                ↓
          File Server (uCore)

---

## Declarative Build

### Base Image

- Base: https://github.com/ublue-os/ucore  
- Custom image: `ghcr.io/<user>/homelab-coreos-minipc:latest`  
- Built with a BlueBuild recipe that:
  - Installs `wireguard-tools`, `docker`, `nfs-utils`, `fail2ban`.
  - Layers `/etc/wireguard/wg0.conf` (template) and systemd units.
  - Declares NFS mounts and a Compose service to start the stack at boot.

---

## Services

### Media & Frontend Apps

- **Plex** — direct port `32400`, hardware transcoding via Intel QuickSync.  
- **Jellyfin** — direct port `8096` (and optional `8920` for HTTPS).  
- **Overseerr** — media request management, accessed via VPS hostname.  
- **Wizarr** — automated Plex/Jellyfin invite handling.  
- **Nextcloud AIO** — personal cloud and groupware.  
- **Immich** — photo and video backup platform.

All of these are defined in a compose file such as `compose/minipc.yml` in this repo.

Example structure (conceptual):

    version: "3.9"
    services:
      plex:
        image: lscr.io/linuxserver/plex
        network_mode: host
        devices:
          - /dev/dri:/dev/dri
        volumes:
          - /mnt/nas-media:/media:ro
          - ./config/plex:/config

      jellyfin:
        image: lscr.io/linuxserver/jellyfin
        ports: ["8096:8096"]
        devices:
          - /dev/dri:/dev/dri
        volumes:
          - /mnt/nas-media:/media:ro
          - ./config/jellyfin:/config

      overseerr:
        image: lscr.io/linuxserver/overseerr
        ports: ["5055:5055"]
        volumes:
          - ./config/overseerr:/config

---

## Networking

### WireGuard

The mini PC acts as a WireGuard server with the following configuration:

- **Server**: `10.253.0.1/24` (NAB9 mini PC)
- **Listen Port**: `51820`
- **Network**: `10.253.0.0/24`

**Configured Peers**:
- LAN-Desktop-Justin: `10.253.0.6/32`
- VPS: `10.253.0.8/32`
- iPhone: `10.253.0.9/32`
- Framework Laptop Justin: `10.253.0.11/32`

Config template is in `config/wireguard/wg0.conf.template`. Use the setup scripts to generate keys and deploy to `/etc/wireguard/wg0.conf`. The service autostarts with `wg-quick@wg0.service`.

### NFS Mounts

Example `/etc/fstab` entries:

    192.168.7.10:/mnt/storage/Media      /mnt/nas-media      nfs  defaults,ro  0 0
    192.168.7.10:/mnt/storage/Nextcloud  /mnt/nas-nextcloud  nfs  defaults,ro  0 0
    192.168.7.10:/mnt/storage/Photos     /mnt/nas-photos     nfs  defaults,ro  0 0

These mounts are used by Plex, Jellyfin, Nextcloud, Immich, etc.

---

## Installation

### Rebase from Existing Fedora Atomic

1. Begin from any Fedora Atomic base (Silverblue/Kinoite/uBlue).
2. Rebase, reboot, then move to the signed image:

   ```bash
   rpm-ostree rebase ostree-unverified-registry:ghcr.io/zoro11031/homelab-coreos-minipc:latest
   systemctl reboot
   rpm-ostree rebase ostree-image-signed:docker://ghcr.io/zoro11031/homelab-coreos-minipc:latest
   systemctl reboot
   ```

### Generate and Install from ISO

Generate the ISO:

```bash
# Generate ISO from a built and published remote image
sudo bluebuild generate-iso --iso-name homelab-coreos-minipc.iso image ghcr.io/zoro11031/homelab-coreos-minipc

# Build image and generate ISO from a local recipe
sudo bluebuild generate-iso --iso-name homelab-coreos-minipc.iso recipe recipe.yml
```

Flash the ISO onto a USB drive (Fedora Media Writer is recommended) and boot it.
- The ISO file should be inside your working directory (wherever you ran the command).

---

## Updates & Rollbacks

### Update the OS

```bash
sudo rpm-ostree upgrade
sudo systemctl reboot
```

### Rollback

If something goes sideways:

```bash
rpm-ostree rollback
```

---

## Security

- Only Plex and Jellyfin ports are forwarded from WAN to the mini PC.  
- All other applications are reached through the VPS (WireGuard + NPM).  
- Fail2ban blocks repeated offenders on SSH and media ports.  
- Secrets (`.env`, WireGuard keys, Cloudflare tokens) are kept out of Git.  
- Firewall rules allow only what the stack actually uses.

---

## Reboot Behavior

On boot:

1. WireGuard connects to the VPS.  
2. NFS mounts are brought up under `/mnt/nas-*`.  
3. The media stack starts via the Compose systemd service.  
4. Optional timers run health checks and log status.

## Setup

### 1. Configure WireGuard

Generate WireGuard keys and configuration:

```bash
cd config/wireguard
./generate-keys.sh    # Generate all keys and create .env
./apply-config.sh     # Generate wg0.conf from template
./export-peer-configs.sh --endpoint your.public.host:51820 \
    --allowed-ips 10.253.0.0/24 --dns 1.1.1.1
```

The `generate-keys.sh` script will:
- Generate server private/public keys
- Generate keys for all 4 peers (Desktop, VPS, iPhone, Laptop)
- Create a `.env` file with all keys
- Store individual key files in `keys/` directory

The `apply-config.sh` script will:
- Read keys from `.env`
- Generate `wg0.conf` from the template
- Output client configuration details

The `export-peer-configs.sh` script will:
- Read keys from the `keys/` directory
- Generate import-ready client configs in `peer-configs/`
- Validate that all required key files exist before writing anything

**Important**: Update the network interface in `wg0.conf.template` if your system doesn't use `eth0`. Common alternatives: `enp1s0`, `eno1`, etc.

### 2. Configure Docker Compose

```bash
cp compose/.env.example compose/.env
# Edit compose/.env with your service-specific configuration
```

### 3. Update NFS Mounts

Update NFS mount IPs in `files/system/etc/systemd/system/*.mount` to match your file server.

### 4. Deploy Services

```bash
cd compose
docker compose -f media.yml -f web.yml -f cloud.yml up -d
```

## Network

- Direct access: Plex (32400), Jellyfin (8096/8920)
- WireGuard VPN: Server on 10.253.0.0/24 for remote access and VPS tunnel
- NFS: Media from 192.168.7.10

## Base Image

Built on `ghcr.io/ublue-os/ucore:latest` with WireGuard, NFS, and Intel media drivers.
