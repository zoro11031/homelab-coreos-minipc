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

- VPS endpoint: `64.23.212.68:51820`  
- Local peer (NAB9): `10.99.0.2/24`  
- VPS peer: `10.99.0.1/24`  

Config lives in `/etc/wireguard/wg0.conf` via BlueBuild and autostarts with `wg-quick@wg0.service`.

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

---

## Philosophy

- **Declarative over click-ops:** The system is described in Git, not in wizards.  
- **Frontend vs backend separation:** This node runs the apps; the other holds the bits.  
- **Immutable host:** No ad-hoc package installs. Everything goes through the image.  
- **Fast rebuilds:** Reflash → rebase → reboot gets you back to the same state.

---

## License

MIT License — see `LICENSE` for details.
