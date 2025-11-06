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

Every CoreOS-family install expects an Ignition configuration on the very first boot.
Without one, you cannot set a password or SSH key for the default `core` user.
Follow the steps below **before** you boot any installer media.

### Prerequisites

- [`butane`](https://coreos.github.io/butane/) in your `$PATH` (used by `transpile.sh`)
- `mkpasswd` (from the `whois` package on Debian/Ubuntu) to generate a password hash
- An SSH key pair on the machine you're using to prepare the config

### Quick Setup

1. Navigate to the Ignition helpers and copy the template:
   ```bash
   cd ignition
   cp config.bu.template config.bu
   ```

2. Generate a password hash for the `core` user:
   ```bash
   ./generate-password-hash.sh
   ```
   Copy the printed yescrypt hash.

3. Edit `config.bu` and replace the placeholders:
   - `YOUR_GOOD_PASSWORD_HASH_HERE` → the hash from step 2
   - `YOUR_SSH_PUB_KEY_HERE` → your SSH public key (`~/.ssh/id_ed25519.pub`, etc.)
   - Adjust hostname, groups, or additional settings if needed

4. Convert the Butane file to Ignition JSON:
   ```bash
   ./transpile.sh config.bu config.ign
   ```
   The script validates that you removed the placeholders and writes `config.ign`.

5. Keep `config.ign` handy for the installation method you plan to use (see below).

**Automatic rebase behavior:** the bundled Butane template adds systemd units that move the host from stock Fedora CoreOS to the signed `ghcr.io/zoro11031/homelab-coreos-minipc:latest` image. Expect **two automatic reboots** after the first boot: one into the unsigned OCI reference and a second into the signed image. This is normal and confirms the autorebase workflow is active.

Using a custom ISO built from the final image (`bluebuild generate-iso ... image ghcr.io/zoro11031/homelab-coreos-minipc`)?
Remove the autorebase units before running `transpile.sh`; otherwise you will sit through two unnecessary reboots.
Detailed instructions live in [`ignition/README.md`](ignition/README.md#disabling-the-automatic-rebase-units).

---

## Installation

### Option A: Install via Fedora CoreOS Live ISO (Recommended)

1. **Finish the Ignition steps above** so you have a customized `config.ign`.

2. **Download the latest Fedora CoreOS installer ISO** from the [official site](https://fedoraproject.org/coreos/download/) or with the containerized helper:
   ```bash
   podman run --security-opt label=disable --pull=always --rm -v "$(pwd)":/data -w /data \
       quay.io/coreos/coreos-installer:release download -s stable -p metal -f iso
   ```

3. **Write the ISO to removable media**:
   - Linux/macOS: `sudo dd if=fedora-coreos.iso of=/dev/sdX bs=4M status=progress oflag=sync`
   - Windows: flash with [Rufus](https://rufus.ie/) using “DD Image” mode

4. **Boot the target machine** from the live ISO and open a shell prompt.

5. **Install to disk using your Ignition file** (mounted from USB, fetched via network, or copied into `/root`):
   ```bash
   sudo coreos-installer install /dev/sda --ignition-file /path/to/config.ign
   ```
   Swap `/dev/sda` for the actual target device (e.g. `/dev/nvme0n1`).
   Use `--ignition-url` if your config is hosted remotely.

6. **Reboot** once the installer reports success:
   ```bash
   sudo reboot
   ```

7. **Let the first boot complete**. Ignition provisions the host, then the autorebase services trigger the two expected reboots described above. After the second restart you land on the signed `homelab-coreos-minipc` image.

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

### Summary of Behavior

| Scenario                   | Ignition Used | Automatic Rebase | Manual Steps           |
|----------------------------|---------------|------------------|------------------------|
| Fresh install via FCOS ISO | ✅            | ✅               | None — fully automated |
| Custom uCore ISO install   | ❌            | N/A              | Already final image    |

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
