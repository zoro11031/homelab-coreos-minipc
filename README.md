# homelab-coreos-minipc

Frontend application node running on NAB9 mini PC with uCore (Ublue CoreOS).

## Services

- **Media:** Plex, Jellyfin (hardware transcoding via Intel QuickSync)
- **Requests:** Overseerr, Wizarr
- **Cloud:** Nextcloud AIO, Immich
- **Monitoring:** Tautulli

## Setup

1. Configure WireGuard: `cp config/wireguard/wg0.conf.template config/wireguard/wg0.conf`
2. Configure compose: `cp compose/.env.example compose/.env`
3. Update NFS mount IPs in `files/system/etc/systemd/system/*.mount`
4. Deploy: `cd compose && docker compose -f media.yml -f web.yml -f cloud.yml up -d`

## Network

- Direct access: Plex (32400), Jellyfin (8096/8920)
- VPS tunnel: Everything else via WireGuard (10.99.0.0/24)
- NFS: Media from 192.168.7.10

## Base Image

Built on `ghcr.io/ublue-os/ucore:latest` with WireGuard, NFS, and Intel media drivers.
