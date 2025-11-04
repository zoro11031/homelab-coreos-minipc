# Services Documentation

Comprehensive guide to all services running on the NAB9 mini PC.

## Overview

The NAB9 runs multiple containerized services organized into three categories:

1. **Media Services** - Streaming and transcoding (Plex, Jellyfin)
2. **Web Services** - Request management and dashboards (Overseerr, Wizarr)
3. **Cloud Services** - Personal cloud and photo backup (Nextcloud, Immich)

## Media Services

### Plex Media Server

**Purpose:** Stream media content to users with hardware transcoding

**Access:**
- Local: http://192.168.7.x:32400/web
- Remote: https://app.plex.tv

**Configuration:**
```yaml
Container: plex
Image: lscr.io/linuxserver/plex:latest
Ports: 32400 (host network mode)
Volumes:
  - Config: /var/lib/containers/appdata/plex
  - Media: /mnt/nas-media (NFS, read-only)
  - Transcode: /var/lib/containers/appdata/plex/transcode
GPU: /dev/dri (Intel QuickSync)
```

**Initial Setup:**
1. Access web interface: http://192.168.7.x:32400/web
2. Sign in with Plex account
3. Add libraries:
   - Movies: /media/Movies
   - TV Shows: /media/TV
   - Music: /media/Music
4. Enable hardware transcoding:
   - Settings → Transcoder
   - Use hardware acceleration: Intel QuickSync
5. Configure remote access:
   - Settings → Remote Access
   - Manually specify port: 32400

**Features:**
- Direct WAN access (port forwarded)
- Hardware transcoding via Intel QuickSync
- Mobile apps for iOS/Android
- Automatic metadata fetching
- User management with separate accounts

### Jellyfin

**Purpose:** Open-source alternative to Plex for media streaming

**Access:**
- Local: http://192.168.7.x:8096
- Remote: http://your-public-ip:8096

**Configuration:**
```yaml
Container: jellyfin
Image: lscr.io/linuxserver/jellyfin:latest
Ports: 8096 (HTTP), 8920 (HTTPS)
Volumes:
  - Config: /var/lib/containers/appdata/jellyfin/config
  - Cache: /var/lib/containers/appdata/jellyfin/cache
  - Media: /mnt/nas-media (NFS, read-only)
GPU: /dev/dri (Intel QuickSync)
Environment:
  - LIBVA_DRIVER_NAME=iHD
```

**Initial Setup:**
1. Access web interface: http://192.168.7.x:8096
2. Create admin account
3. Add libraries:
   - Movies: /media/Movies
   - TV Shows: /media/TV
   - Music: /media/Music
4. Enable hardware acceleration:
   - Dashboard → Playback
   - Hardware acceleration: Intel QuickSync
   - Enable H.264, HEVC codecs

**Features:**
- Fully open-source
- No premium subscriptions
- Hardware transcoding
- Live TV and DVR support (with tuner)
- Plugin system

### Tautulli

**Purpose:** Plex monitoring and statistics

**Access:** http://192.168.7.x:8181

**Configuration:**
```yaml
Container: tautulli
Image: lscr.io/linuxserver/tautulli:latest
Ports: 8181
Volumes:
  - Config: /var/lib/containers/appdata/tautulli
```

**Features:**
- Track Plex usage statistics
- User activity monitoring
- Bandwidth monitoring
- Custom notifications
- Integration with Discord, Telegram, etc.

## Web Services

### Overseerr

**Purpose:** Media request management for Plex/Jellyfin

**Access:**
- Local: http://192.168.7.x:5055
- Remote: https://overseerr.yourdomain.com (via VPS)

**Configuration:**
```yaml
Container: overseerr
Image: lscr.io/linuxserver/overseerr:latest
Ports: 5055
Volumes:
  - Config: /var/lib/containers/appdata/overseerr
```

**Initial Setup:**
1. Access web interface
2. Sign in with Plex account
3. Connect to Plex/Jellyfin servers
4. Configure request workflows
5. Set up user permissions

**Features:**
- Users can request movies and TV shows
- Approval workflows
- Integration with Sonarr/Radarr (on file server)
- User notifications
- Discovery interface

### Wizarr

**Purpose:** Automated user invitation system

**Access:**
- Local: http://192.168.7.x:5690
- Remote: https://invite.yourdomain.com (via VPS)

**Configuration:**
```yaml
Container: wizarr
Image: ghcr.io/wizarrrr/wizarr:latest
Ports: 5690
Volumes:
  - Config: /var/lib/containers/appdata/wizarr
```

**Features:**
- Generate invitation links for Plex/Jellyfin
- Automatic user onboarding
- Custom invitation pages
- Time-limited invitations
- User analytics

### Organizr (Optional)

**Purpose:** Unified dashboard for all services

**Access:** http://192.168.7.x:9983

**Configuration:**
```yaml
Container: organizr
Image: organizr/organizr:latest
Ports: 9983
Volumes:
  - Config: /var/lib/containers/appdata/organizr
```

**Features:**
- Single interface for all services
- Tab-based organization
- SSO integration
- Custom themes
- Homepage customization

### Homepage (Optional)

**Purpose:** Modern service dashboard

**Access:** http://192.168.7.x:3000

**Configuration:**
```yaml
Container: homepage
Image: ghcr.io/gethomepage/homepage:latest
Ports: 3000
Volumes:
  - Config: /var/lib/containers/appdata/homepage
  - Docker socket: /var/run/docker.sock (read-only)
```

**Features:**
- Real-time service status
- Docker integration
- Weather widgets
- Bookmarks
- Search integration

## Cloud Services

### Nextcloud All-in-One

**Purpose:** Personal cloud, file storage, and collaboration

**Access:**
- Local: http://192.168.7.x:8080 (admin), http://192.168.7.x:11000 (client)
- Remote: https://cloud.yourdomain.com (via VPS)

**Configuration:**
```yaml
Container: nextcloud-aio-mastercontainer
Image: nextcloud/all-in-one:latest
Ports: 8080 (admin), 11000 (Apache)
Volumes:
  - Config: nextcloud_aio_mastercontainer
  - Data: /mnt/nas-nextcloud (NFS)
  - Docker socket: /var/run/docker.sock
```

**Initial Setup:**
1. Access admin interface: http://192.168.7.x:8080
2. Set admin password
3. Configure domain name
4. Enable/disable optional containers
5. Start Nextcloud
6. Access client: http://192.168.7.x:11000

**Features:**
- File sync and share
- Calendar and contacts
- Office documents (Collabora/OnlyOffice)
- Photo gallery
- Video calls (Talk)
- Notes and tasks
- Mobile apps

**Included Containers (managed by AIO):**
- Nextcloud (main application)
- PostgreSQL (database)
- Redis (caching)
- Collabora/OnlyOffice (document editing)
- Talk (video conferencing)
- Imaginary (image processing)
- ClamAV (virus scanning)

### Immich

**Purpose:** Self-hosted photo and video backup

**Access:**
- Local: http://192.168.7.x:2283
- Remote: https://photos.yourdomain.com (via VPS)

**Configuration:**
```yaml
Services:
  - immich-server (main API)
  - immich-microservices (background tasks)
  - immich-machine-learning (AI features)
  - immich-postgres (database)
  - immich-redis (caching)

Main Container: immich-server
Image: ghcr.io/immich-app/immich-server:release
Ports: 2283
Volumes:
  - Photos: /mnt/nas-immich (NFS)
  - Database: /var/lib/containers/appdata/immich/postgres
  - ML Cache: /var/lib/containers/appdata/immich/model-cache
```

**Initial Setup:**
1. Access web interface: http://192.168.7.x:2283
2. Create admin account
3. Install mobile app (iOS/Android)
4. Log in to mobile app
5. Enable automatic backup

**Features:**
- Automatic photo/video backup from mobile
- AI-powered facial recognition
- Object and scene detection
- Photo search
- Albums and sharing
- Timeline view
- RAW photo support
- EXIF metadata preservation

## Service Management

### Starting/Stopping Services

```bash
# All services
cd ~/homelab-coreos-minipc/compose
docker compose -f media.yml -f web.yml -f cloud.yml up -d
docker compose -f media.yml -f web.yml -f cloud.yml down

# Individual service groups
docker compose -f media.yml up -d
docker compose -f web.yml up -d
docker compose -f cloud.yml up -d

# Individual services
docker stop plex
docker start plex
docker restart jellyfin
```

### Viewing Logs

```bash
# All services in a compose file
docker compose -f media.yml logs -f

# Specific service
docker logs -f plex
docker logs --tail 100 jellyfin
```

### Updating Services

```bash
# Pull latest images
docker compose -f media.yml -f web.yml -f cloud.yml pull

# Recreate containers with new images
docker compose -f media.yml -f web.yml -f cloud.yml up -d

# Or update specific service
docker compose -f media.yml pull plex
docker compose -f media.yml up -d plex
```

### Resource Usage

```bash
# View container resource usage
docker stats

# View specific container
docker stats plex jellyfin

# Check disk usage
docker system df
```

## Backup Recommendations

### Configuration Backups

All container configurations are stored in:
```
/var/lib/containers/appdata/
├── plex/
├── jellyfin/
├── overseerr/
├── wizarr/
├── nextcloud/ (AIO managed)
├── immich/
└── ...
```

**Backup Strategy:**
1. Stop containers
2. Backup entire appdata directory
3. Backup Docker Compose .env file
4. Backup WireGuard configuration
5. Restart containers

**Automated Backup Script:**
```bash
#!/bin/bash
systemctl stop docker
tar -czf /backup/appdata-$(date +%Y%m%d).tar.gz /var/lib/containers/appdata
systemctl start docker
```

### Database Backups

- **Immich:** Backup PostgreSQL database
  ```bash
  docker exec immich-postgres pg_dumpall -U postgres > immich-backup.sql
  ```

- **Nextcloud:** Use built-in backup (managed by AIO)

### Media Backups

Media files are stored on the NFS file server, which should have its own backup strategy.

## Monitoring

### Health Checks

Use the provided scripts:

```bash
# NFS mount health
~/homelab-coreos-minipc/scripts/nfs-health.sh

# WireGuard connectivity
~/homelab-coreos-minipc/scripts/wireguard-check.sh

# GPU functionality
~/homelab-coreos-minipc/scripts/gpu-verify.sh
```

### Container Health

```bash
# Check container status
docker ps -a

# Check container health
docker inspect plex | grep -A 10 Health

# Monitor logs for errors
docker compose -f media.yml logs --tail 100 | grep -i error
```

## Security Considerations

1. **Network Isolation**
   - Only Plex and Jellyfin are directly exposed to WAN
   - Other services accessed via VPS reverse proxy

2. **Authentication**
   - Enable 2FA on Nextcloud and Immich
   - Use strong passwords for all services
   - Configure Plex/Jellyfin user accounts

3. **Updates**
   - Keep containers updated
   - Monitor security advisories
   - Use Watchtower for automatic updates

4. **Firewall**
   - UFW configured to block unnecessary ports
   - Only allow traffic from local network and WireGuard

5. **Secrets Management**
   - Never commit .env file to git
   - Secure WireGuard private keys
   - Use strong database passwords

## Performance Tuning

### Plex Transcoding

- Use local SSD for transcode temp directory
- Enable hardware transcoding
- Optimize library scanning (schedule during off-hours)

### Jellyfin Transcoding

- Configure transcode path to local SSD
- Enable Intel QuickSync
- Adjust transcode quality settings

### NFS Performance

- Use TCP instead of UDP for reliability
- Large rsize/wsize (131072) for better throughput
- Consider jumbo frames on local network

### Container Resources

Set resource limits in compose files if needed:

```yaml
deploy:
  resources:
    limits:
      memory: 4G
    reservations:
      memory: 2G
```
