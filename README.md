# Homelab CoreOS Mini PC

Declarative image + helper tooling for the NAB9 mini PC that fronts my homelab. It rebases Fedora CoreOS into a custom UBlue uCore build, tunnels traffic through WireGuard to a VPS, and mounts media from the backend file server over NFS.

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
- [`docs/getting-started.md`](docs/getting-started.md): consolidated setup walkthrough covering prerequisites, menu options, and manual fallbacks.
- [`docs/reference/ignition.md`](docs/reference/ignition.md): Butane/Ignition workflow plus image-layering details.
- [`docs/reference/homelab-setup-cli.md`](docs/reference/homelab-setup-cli.md): full manual for the helper scripts and CLI.
- [`docs/testing/virt-manager-qa.md`](docs/testing/virt-manager-qa.md): virt-manager smoke + regression checklist.
- [`docs/operations/weekend-deployment.md`](docs/operations/weekend-deployment.md): release/rollback checklist.
- [`docs/audits/2025-go-audit.md`](docs/audits/2025-go-audit.md) and [`docs/audits/2025-go-audit-changelog.md`](docs/audits/2025-go-audit-changelog.md): audit records and fix history.
- [`docs/dev`](docs/dev): devcontainer, CI build pipeline, and final review checklist.
- [`docs/roadmap`](docs/roadmap): rewrite plan, backlog, and phase handoffs.

This repo is intentionally blunt. The docs highlight what I need when rebuilding the node, and everything else lives in the code or compose templates.
