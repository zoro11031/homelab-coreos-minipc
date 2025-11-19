# Homelab CoreOS Mini PC &nbsp; [![build](https://github.com/zoro11031/homelab-coreos-minipc/actions/workflows/build.yml/badge.svg)](https://github.com/zoro11031/homelab-coreos-minipc/actions/workflows/build.yml)

Declarative image + helper tooling for the NAB9 mini PC that fronts my homelab. It rebases Fedora CoreOS into a custom UBlue uCore build, tunnels traffic through WireGuard to a VPS, and mounts media from the backend file server over NFS.

## Scope & assumptions
- Single-node helper meant for my own homelab. If you grab it, expect "works on my LAN" defaults.
- Focuses on the interactive Go helper (menu-based) with optional fallbacks to the legacy bash scripts under `files/`.
- Inputs are trusted. The wizard validates obvious pitfalls but intentionally avoids enterprise-grade policy layers.
- **Note:** The `homelab-setup` Go CLI source code is maintained in a separate repository at [plex-migration-homelab/homelab-setup](https://github.com/plex-migration-homelab/homelab-setup). This repo contains only the compiled binary in `files/system/usr/bin/homelab-setup`, which is automatically rebuilt by GitHub Actions when changes are pushed to the upstream repo.


## What's running
- **Media:** Plex and Jellyfin with Intel QuickSync for hardware transcodes.
- **Portals:** Overseerr, Wizarr, and Nginx Proxy Manager on the VPS for public access.
- **Cloud:** Nextcloud + Collabora + Redis + PostgreSQL and Immich for photos.
- **Platform bits baked into the image:** Podman, systemd units for WireGuard/NFS/compose, helper scripts dropped into `~/setup/`, and VAAPI drivers so the box is ready for GPU work.

## Try it in a weekend
1. **Install the image.** Build an Ignition file from `ignition/config.bu.template` (see [`docs/reference/ignition.md`](docs/reference/ignition.md)) and install Fedora CoreOS on the target mini PC. The first boot rebases into `ghcr.io/zoro11031/homelab-coreos-minipc`.
2. **Run the helper.** SSH in as `core`, jump into `~/setup/home-lab-setup-scripts/`, and launch `homelab-setup` (Go CLI) or `./homelab-setup.sh` (legacy bash). The wizard walks through user creation, WireGuard, NFS mounts, compose secrets, and service deployment.
3. **Expose services.** Plex/Jellyfin stay on direct ports. Everything else routes through the VPS via WireGuard + Nginx Proxy Manager. Config files for tunnels and compose stacks are under `~/setup/` and `/srv/containers/`.

## Documentation map
- [`docs/getting-started.md`](docs/getting-started.md): walkthrough for the image install plus the Go helper menu.
- [`docs/reference/ignition.md`](docs/reference/ignition.md): Butane/Ignition workflow and image layering details.
- [`docs/reference/homelab-setup-cli.md`](docs/reference/homelab-setup-cli.md): legacy bash script reference kept for historical context; the Go helper re-uses the same concepts but is menu-driven only.
- [`docs/testing/virt-manager-qa.md`](docs/testing/virt-manager-qa.md): quick smoke checklist for the VM validation flow.
