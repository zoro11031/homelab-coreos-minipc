#!/bin/bash
# UFW Firewall Configuration for NAB9 Mini PC
# This script sets up firewall rules for the frontend application node

set -euo pipefail

echo "=== Configuring UFW Firewall ==="
echo

# Reset UFW to default state
echo "Resetting UFW to default configuration..."
sudo ufw --force reset

# Set default policies
echo "Setting default policies..."
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (IMPORTANT: Enable before enabling UFW!)
echo "Allowing SSH (port 22)..."
sudo ufw allow 22/tcp comment 'SSH'

# Allow Plex direct access
echo "Allowing Plex (port 32400)..."
sudo ufw allow 32400/tcp comment 'Plex Media Server'

# Allow Jellyfin direct access
echo "Allowing Jellyfin (ports 8096, 8920)..."
sudo ufw allow 8096/tcp comment 'Jellyfin HTTP'
sudo ufw allow 8920/tcp comment 'Jellyfin HTTPS'

# Allow WireGuard
echo "Allowing WireGuard (port 51820)..."
sudo ufw allow 51820/udp comment 'WireGuard VPN'

# Allow local network (192.168.7.0/24) - for NFS and local management
echo "Allowing local network traffic (192.168.7.0/24)..."
sudo ufw allow from 192.168.7.0/24 comment 'Local network'

# Allow WireGuard tunnel network (10.99.0.0/24)
echo "Allowing WireGuard tunnel network (10.99.0.0/24)..."
sudo ufw allow from 10.99.0.0/24 comment 'WireGuard tunnel'

# Rate limit SSH to prevent brute force
echo "Configuring SSH rate limiting..."
sudo ufw limit 22/tcp comment 'SSH rate limit'

# Enable UFW
echo
echo "Enabling UFW firewall..."
sudo ufw --force enable

# Show status
echo
echo "=== Firewall Status ==="
sudo ufw status verbose

echo
echo "=== Firewall Configuration Complete ==="
echo
echo "Open ports:"
echo "  - 22/tcp   : SSH (rate limited)"
echo "  - 32400/tcp: Plex Media Server"
echo "  - 8096/tcp : Jellyfin HTTP"
echo "  - 8920/tcp : Jellyfin HTTPS"
echo "  - 51820/udp: WireGuard VPN"
echo
echo "Allowed networks:"
echo "  - 192.168.7.0/24 : Local LAN"
echo "  - 10.99.0.0/24   : WireGuard tunnel"
echo
echo "IMPORTANT: Other services (Overseerr, Nextcloud, Immich) are only"
echo "accessible via the WireGuard tunnel and VPS reverse proxy."
