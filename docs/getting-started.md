# Getting Started

Quick, opinionated walkthrough for rebuilding the NAB9 mini PC. Use it when you want to go from "fresh CoreOS install" to "services are back" without digging through every README.

## Before you plug cables back in
- Install Fedora CoreOS with the Ignition file generated from `ignition/config.bu.template` (see [`docs/reference/ignition.md`](reference/ignition.md)).
- SSH in as the `core` user and confirm you can reach the file server (if you mount media via NFS).
- Make sure the image has `podman`, `podman-compose`, `nfs-utils`, `wireguard-tools`, and systemd units from this repo.
- Collect VPN details (WireGuard peers, VPS port) and any passwords you plan to reuse.

## Fire up the helper scripts
After first boot the image drops everything under `~/setup/`.

```bash
cd ~/setup/home-lab-setup-scripts
homelab-setup            # Go CLI (recommended)
# or
./homelab-setup.sh       # legacy bash wrapper
```

The menu gives you:

```
[A] Run All Steps (complete setup)
[Q] Quick Setup (skip WireGuard)
[0-6] Run a specific step
[T] Troubleshooting menu
[S] Show setup status
[P] Add WireGuard Peer (post-setup)
```

### What the wizard configures
1. **Runtime + users:** pick Podman or Docker, optionally create a dedicated container user, and wire up UID/GID/subuid/subgid.
2. **Directories:** `/srv/containers/{media,web,cloud}`, `/var/lib/containers/appdata`, `/mnt/nas-*` mount points, and compose/env scaffolding.
3. **WireGuard:** key generation, interface templates, peer exports, and `wg-quick@wg0` units. After the initial setup you can return to the menu (or run `homelab-setup wireguard add-peer`) to append additional peers. Each run writes a sanitized `[Peer]` block to `/etc/wireguard/<iface>.conf`, exports a ready-to-import client file under `~/setup/export/wireguard-peers/`, and prints both the config text and an ASCII QR code for mobile imports (install `qrencode` to enable the QR renderer).
4. **NFS mounts:** server detection, mount tests, and systemd unit enablement.
5. **Services:** copies compose files, collects Plex claims + service passwords, writes `.env` files, and launches the Podman stack via systemd.
6. **Status + troubleshooting:** baked-in diagnostics and reset commands (`homelab-setup status|reset|troubleshoot`).

Configuration lives in `~/.homelab-setup.conf`, so you can rerun the wizard without retyping defaults.

## Manual checklist
Prefer to drive everything yourself? Run through these steps:

1. **Users + permissions:** create a service account (e.g., `containeruser`), add it to `wheel`, and grant `/etc/subuid` + `/etc/subgid` ranges.
2. **Directory layout:** build `/srv/containers/` for compose files and `/var/lib/containers/appdata/` for persistent data. Ensure ownership matches the service account.
3. **WireGuard:** use `config/wireguard/wg0.conf.template`, drop the final config in `/etc/wireguard/wg0.conf`, and enable `systemctl enable --now wg-quick@wg0`.
4. **NFS mounts:** add exports to `/etc/fstab` or install the provided `.mount` units under `/etc/systemd/system/`.
5. **Compose stacks:** copy templates from `~/setup/compose-setup/`, edit `.env` files with secrets, and start the systemd units (`podman-compose-media.service`, etc.).
6. **Verification:** `podman ps`, `sudo systemctl status podman-compose-*.service`, and `mount | grep /mnt/nas` should all look clean.

## Common passwords + URLs
| Service | Default port | Notes |
| --- | --- | --- |
| Plex | `http://<ip>:32400/web` | Claim token required during setup |
| Jellyfin | `http://<ip>:8096` | TLS via `https://<ip>:8920` if desired |
| Overseerr | `http://<ip>:5055` | Routed through the VPS in production |
| Nextcloud | `http://<ip>:8080` | Admin + DB passwords prompted by wizard |
| Immich | `http://<ip>:2283` | PostgreSQL + Redis secrets set during setup |

## Troubleshooting cheats
```bash
homelab-setup troubleshoot        # Run the bundled diagnostics
sudo journalctl -u podman-compose-media.service -f
sudo systemctl status mnt-nas-media.mount
wg show
```

If something goes sideways, rerun the relevant step from the menu or delete `.homelab-setup.conf` to start fresh. Everything is idempotent—the helper will re-apply settings without nuking running containers unless you tell it to reset.
