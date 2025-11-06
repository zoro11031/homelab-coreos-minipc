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
| WireGuard               | Connects to DigitalOcean VPS            |
| Nginx Proxy Manager     | SSL / reverse proxy on VPS side         |
| Fail2ban + firewall     | Protects exposed media / SSH ports      |
| NFS client              | Mounts media from the file server       |

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
│  ┌──────────────────┐  │                     │
│  │ Nginx Proxy      │  │                     │
│  │ Manager          │  │                     │
│  │ (SSL/reverse     │  │                     │
│  │  proxy)          │  │                     │
│  │ Routes:          │  │                     │
│  │ • Overseerr      │  │                     │
│  │ • Wizarr         │  │                     │
│  │ • Nextcloud      │  │                     │
│  │ • Immich         │  │                     │
│  └──────────────────┘                      │
             |                                 |
             |                                 │                                  
  WireGuard Tunnel (encrypted)                 │
             │                                 │
             ↓                                 │
   ┌────────────────────────────────────────────┴─────────────┐
   │                NAB9 Mini PC (Ublue CoreOS)               │
   │                (Frontend / User-Facing)                  │
   │  ┌────────────────────────────────────────────────────┐  │
   │  │                                                    │  │
   │  │ • Docker Stack                                     │  │
   │  │ • Jellyfin (direct exposure)                       │  │
   │  │ • Plex (direct exposure)                           │  │
   │  │ • Overseerr / Wizarr / Nextcloud / Immich (via     │  │
   │  │   VPS)                                             │  │
   │  └────────────────────────────────────────────────────┘  │
   └───────────────┬──────────────────────────────────────────┘
                   │
             (Direct LAN Connection)
                   │
                   ↓
      ┌─────────────────────────────────────────┐
      │      File Server (Ublue CoreOS)         │
      │      (Backend / Storage & Automation)   │
      │  ┌───────────────────────────────────┐  │
      │  │ • SnapRAID + mergerfs             │  │
      │  │ • NFS Server (192.168.7.x)        │  │
      │  │ • Sonarr / Radarr / Lidarr        │  │
      │  │ • Prowlarr                        │  │
      │  │ • qBittorrent + VPN (outbound)    │  │
      │  │ • Outbound downloads via pfSense  │  │
      │  │ • No inbound exposure (LAN only)  │  │
      │  └───────────────────────────────────┘  │
      └─────────────────────────────────────────┘
                   │
                   ↓
              DAS Storage
        (18-19 data disks + parity)
```

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
- **Nextcloud** — personal cloud and groupware with PostgreSQL, Redis, and Collabora.  
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

Config template is in `config/wireguard/wg0.conf.template`. Use the setup scripts in `files/setup_scripts/` to generate keys and deploy to `/etc/wireguard/wg0.conf`. The service autostarts with `wg-quick@wg0.service`.

### NFS Mounts

Example `/etc/fstab` entries:

    192.168.7.10:/mnt/storage/Media      /mnt/nas-media      nfs  defaults,ro  0 0
    192.168.7.10:/mnt/storage/Nextcloud  /mnt/nas-nextcloud  nfs  defaults,ro  0 0
    192.168.7.10:/mnt/storage/Photos     /mnt/nas-photos     nfs  defaults,ro  0 0

These mounts are used by Plex, Jellyfin, Nextcloud, Immich, etc.

---

## Ignition Setup (First-Time Installation)

**All CoreOS-family installations require an Ignition file** to configure the system on first boot.
At minimum, you must set a password and SSH key for the default `core` user.

### Quick Setup

1. Navigate to the `ignition/` directory:
   ```bash
   cd ignition
   ```

2. Copy the template:
   ```bash
   cp config.bu.template config.bu
   ```

3. Generate a password hash:
   ```bash
   ./generate-password-hash.sh
   ```

4. Edit `config.bu` and update:
   - `YOUR_GOOD_PASSWORD_HASH_HERE` → your generated password hash
   - `YOUR_SSH_PUB_KEY_HERE` → your SSH public key (e.g. `~/.ssh/id_ed25519.pub`)

5. Transpile to Ignition JSON:
   ```bash
   ./transpile.sh config.bu config.ign
   ```

6. Use `config.ign` during installation (see methods below).

**Note:**
This Ignition file includes automatic rebase services that transition the system from the base CoreOS image to your custom uCore image.
Your system will reboot **twice** after the first boot — once to move to the unsigned OCI image, and once more to the signed OCI image.
This behavior is **expected** and indicates the autorebase process is working.

If you plan to generate a uCore ISO directly from the built container image (using `bluebuild generate-iso`), **remove** these autorebase units before transpiling.
In that case, the installed image already boots into the target, and the autorebase units would cause redundant reboots.
See [`ignition/README.md`](ignition/README.md#disabling-the-automatic-rebase-units) for details.

---

## Installation

### Option A: Install via Fedora CoreOS Live ISO (Recommended)

1. Prepare your Ignition file (`config.ign`) as shown above.

2. Download a **Fedora CoreOS live ISO** (not a uCore ISO):
   [https://fedoraproject.org/coreos/download/](https://fedoraproject.org/coreos/download/)

3. Boot from the ISO and install uCore with Ignition:
   ```bash
   sudo coreos-installer install /dev/sdX \
     --image-url https://github.com/ublue-os/ucore/releases/download/latest/ucore-x86_64.raw.xz \
     --ignition-file ignition/config.ign
   ```
   Replace `/dev/sdX` with your target drive (e.g. `/dev/nvme0n1`).

4. On first boot:
   - The system applies your Ignition configuration.
   - Then it automatically rebases twice:
     - First to the **unsigned** OCI image.
     - Then to the **signed** OCI image.
   - After both reboots, it will be running your final, signed `homelab-coreos-minipc` image.

---

### Option B: Generate a Custom ISO (Advanced Workflow)

If you want a standalone ISO that already contains your custom image:

1. Build or reference your image:
   ```bash
   sudo bluebuild generate-iso --iso-name homelab-coreos-minipc.iso \
     image ghcr.io/zoro11031/homelab-coreos-minipc
   ```

   Or from a local recipe:
   ```bash
   sudo bluebuild generate-iso --iso-name homelab-coreos-minipc.iso \
     recipe recipe.yml
   ```

2. **Do not use `coreos-installer iso ignition embed`** on uCore or uBlue-generated ISOs —
   they are not CoreOS live ISOs and do not support embedding.
   Use the installer-based Ignition method from Option A instead.

3. Flash your ISO with Fedora Media Writer, Ventoy, or `dd`, and boot.

---

### Option C: Rebase Manually (Alternate Path)

If you are already running Fedora Atomic (Silverblue, Kinoite, or uBlue) and don't want to reinstall:

```bash
rpm-ostree rebase ostree-unverified-registry:ghcr.io/zoro11031/homelab-coreos-minipc:latest
systemctl reboot
# After reboot, finalize to signed image
rpm-ostree rebase ostree-image-signed:docker://ghcr.io/zoro11031/homelab-coreos-minipc:latest
systemctl reboot
```

This path skips Ignition entirely and is for converting an existing installation.

---

### Summary of Behavior

| Scenario                   | Ignition Used | Automatic Rebase | Manual Steps           |
|----------------------------|---------------|------------------|------------------------|
| Fresh install via FCOS ISO | ✅            | ✅               | None — fully automated |
| Custom uCore ISO install   | ❌            | N/A              | Already final image    |
| Existing Fedora Atomic     | ❌            | ❌               | Manual rebase required |

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
