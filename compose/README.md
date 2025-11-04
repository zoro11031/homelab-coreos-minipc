# Docker Compose Services

This directory contains Docker Compose configurations for all services running on the NAB9 mini PC.

## Service Organization

Services are organized into three logical groups:

### media.yml
Media streaming services with hardware transcoding:
- **Plex** (port 32400) - Direct WAN access, Intel QuickSync transcoding
- **Jellyfin** (ports 8096, 8920) - Direct WAN access, Intel QuickSync transcoding
- **Tautulli** (port 8181) - Plex monitoring and statistics

### web.yml
Web-based management and request services (accessed via VPS):
- **Overseerr** (port 5055) - Media request management
- **Wizarr** (port 5690) - Automated user invitations
- **Organizr** (port 9983) - Unified dashboard (optional)
- **Homepage** (port 3000) - Service dashboard (optional)

### cloud.yml
Personal cloud and photo services (accessed via VPS):
- **Nextcloud AIO** (ports 8080, 11000) - Personal cloud and groupware
- **Immich** (port 2283) - Photo and video backup platform

## Quick Start

### 1. Configure Environment Variables

```bash
cd compose/
cp .env.example .env
nano .env
```

Fill in all required values, especially:
- Plex claim token (from https://www.plex.tv/claim/)
- Immich database password
- Your timezone

### 2. Create Required Directories

```bash
sudo mkdir -p /var/lib/containers/appdata/{plex,jellyfin,tautulli,overseerr,wizarr,nextcloud,immich,organizr,homepage}
sudo mkdir -p /var/lib/containers/appdata/immich/{postgres,model-cache}
```

### 3. Verify NFS Mounts

Ensure NFS mounts are active before starting services:

```bash
sudo systemctl status mnt-nas-media.mount
sudo systemctl status mnt-nas-nextcloud.mount
sudo systemctl status mnt-nas-immich.mount
```

### 4. Start Services

Start all services:
```bash
docker compose -f media.yml -f web.yml -f cloud.yml up -d
```

Or start individual service groups:
```bash
docker compose -f media.yml up -d
docker compose -f web.yml up -d
docker compose -f cloud.yml up -d
```

## Service Management

### View Logs
```bash
docker compose -f media.yml logs -f
docker logs plex
docker logs jellyfin
```

### Stop Services
```bash
docker compose -f media.yml -f web.yml -f cloud.yml down
```

### Update Services
```bash
docker compose -f media.yml -f web.yml -f cloud.yml pull
docker compose -f media.yml -f web.yml -f cloud.yml up -d
```

## Hardware Transcoding

Both Plex and Jellyfin are configured for Intel QuickSync hardware transcoding:

- Device mapping: `/dev/dri:/dev/dri`
- Environment: `LIBVA_DRIVER_NAME=iHD` (Jellyfin)
- Tested with Intel 12th gen+ processors

Verify hardware transcoding is working:
```bash
# Check if GPU is accessible to containers
docker exec plex ls -l /dev/dri
docker exec jellyfin vainfo
```

## Network Access

### Direct WAN Access
- Plex: Port 32400 (configured in router port forwarding)
- Jellyfin: Ports 8096, 8920 (configured in router port forwarding)

### VPS Reverse Proxy Access (via WireGuard)
- Overseerr: http://10.99.0.2:5055
- Wizarr: http://10.99.0.2:5690
- Nextcloud: http://10.99.0.2:11000
- Immich: http://10.99.0.2:2283

Configure Nginx Proxy Manager on VPS to route traffic to these internal addresses.

## Storage Paths

### NFS Mounts (from file server)
- Media: `/mnt/nas-media` (read-only)
- Nextcloud data: `/mnt/nas-nextcloud` (read-write)
- Immich photos: `/mnt/nas-immich` (read-write)

### Local Storage (container configs and databases)
- Base path: `/var/lib/containers/appdata/`
- Each service has its own subdirectory

## Troubleshooting

### Services won't start
1. Check NFS mounts are active
2. Verify `.env` file exists and is configured
3. Check Docker is running: `systemctl status docker`

### Hardware transcoding not working
1. Run GPU setup script: `sudo ./config/gpu/intel-qsv-setup.sh`
2. Verify GPU access: `ls -l /dev/dri/`
3. Check container has GPU device: `docker exec plex ls -l /dev/dri`

### Can't access services via VPS
1. Check WireGuard is connected: `sudo wg show`
2. Verify firewall allows WireGuard: `sudo ufw status`
3. Test connectivity from VPS: `ping 10.99.0.2`

## Automatic Updates

Services are labeled for Watchtower automatic updates (except Nextcloud AIO).

To enable Watchtower:
```bash
docker run -d \
  --name watchtower \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --cleanup \
  --interval 86400
```

## Security Notes

- Never commit `.env` file to git (it's in `.gitignore`)
- All services except Plex/Jellyfin should only be accessible via VPS
- Keep WireGuard configuration secure
- Use strong passwords for database services
- Enable two-factor authentication where available (Nextcloud, Immich)
