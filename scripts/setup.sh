#!/bin/bash
# Initial System Setup Script for NAB9 Mini PC
# This script performs the initial configuration after OS installation

set -euo pipefail

echo "╔════════════════════════════════════════════════════════════╗"
echo "║  NAB9 Mini PC - Initial System Setup                      ║"
echo "║  Frontend Application Node Configuration                  ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root (use sudo)"
    exit 1
fi

# Confirm before proceeding
read -p "This will configure WireGuard, NFS, firewall, and GPU. Continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Setup cancelled."
    exit 0
fi

echo "=== Step 1: System Information ==="
echo "Hostname: $(hostname)"
echo "OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo "Kernel: $(uname -r)"
echo "Architecture: $(uname -m)"
echo

# Detect repo location
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Repository location: $REPO_DIR"
echo

echo "=== Step 2: Installing WireGuard Configuration ==="
if [ ! -f "$REPO_DIR/config/wireguard/wg0.conf" ]; then
    echo "ERROR: WireGuard config not found at $REPO_DIR/config/wireguard/wg0.conf"
    echo "Copy wg0.conf.template to wg0.conf and fill in your keys first!"
    exit 1
fi

echo "Installing WireGuard configuration..."
mkdir -p /etc/wireguard
cp "$REPO_DIR/config/wireguard/wg0.conf" /etc/wireguard/wg0.conf
chmod 600 /etc/wireguard/wg0.conf
echo "✓ WireGuard config installed"

echo "Starting WireGuard..."
systemctl enable wg-quick@wg0.service
systemctl restart wg-quick@wg0.service
sleep 2
if systemctl is-active --quiet wg-quick@wg0.service; then
    echo "✓ WireGuard is running"
    wg show
else
    echo "⚠ WARNING: WireGuard failed to start. Check configuration."
fi
echo

echo "=== Step 3: Creating Mount Points ==="
mkdir -p /mnt/nas-media /mnt/nas-nextcloud /mnt/nas-immich
echo "✓ Mount points created"
echo

echo "=== Step 4: Configuring NFS Mounts ==="
echo "Enabling NFS mount units..."
systemctl enable mnt-nas-media.mount
systemctl enable mnt-nas-nextcloud.mount
systemctl enable mnt-nas-immich.mount

echo "Starting NFS mounts..."
systemctl start mnt-nas-media.mount || echo "⚠ Failed to mount /mnt/nas-media"
systemctl start mnt-nas-nextcloud.mount || echo "⚠ Failed to mount /mnt/nas-nextcloud"
systemctl start mnt-nas-immich.mount || echo "⚠ Failed to mount /mnt/nas-immich"

echo
echo "NFS mount status:"
systemctl status mnt-nas-media.mount --no-pager | grep Active
systemctl status mnt-nas-nextcloud.mount --no-pager | grep Active
systemctl status mnt-nas-immich.mount --no-pager | grep Active
echo

echo "=== Step 5: Configuring Firewall ==="
if command -v ufw &> /dev/null; then
    echo "Running UFW firewall configuration..."
    "$REPO_DIR/config/firewall/ufw-rules.sh"
    echo "✓ Firewall configured"
else
    echo "⚠ WARNING: UFW not found. Install ufw package."
fi
echo

echo "=== Step 6: Configuring GPU (Intel QuickSync) ==="
if [ -f "$REPO_DIR/config/gpu/intel-qsv-setup.sh" ]; then
    "$REPO_DIR/config/gpu/intel-qsv-setup.sh"
else
    echo "⚠ GPU setup script not found"
fi
echo

echo "=== Step 7: Creating Application Data Directories ==="
APPDATA_PATH="/var/lib/containers/appdata"
mkdir -p "$APPDATA_PATH"/{plex,jellyfin,tautulli,overseerr,wizarr,nextcloud,immich,organizr,homepage}
mkdir -p "$APPDATA_PATH/immich"/{postgres,model-cache}
mkdir -p "$APPDATA_PATH/plex/transcode"

echo "✓ Application directories created at $APPDATA_PATH"
echo

echo "=== Step 8: Docker Service ==="
if command -v docker &> /dev/null; then
    echo "Docker version: $(docker --version)"
    systemctl enable docker.service
    systemctl start docker.service
    echo "✓ Docker service enabled and started"
else
    echo "⚠ WARNING: Docker not found. It will be installed on next boot from the image."
fi
echo

echo "=== Step 9: Configure Docker Compose Environment ==="
if [ ! -f "$REPO_DIR/compose/.env" ]; then
    echo "⚠ WARNING: $REPO_DIR/compose/.env not found"
    echo "Copy .env.example to .env and configure before starting services:"
    echo "  cd $REPO_DIR/compose"
    echo "  cp .env.example .env"
    echo "  nano .env"
else
    echo "✓ Docker Compose .env file exists"
fi
echo

echo "╔════════════════════════════════════════════════════════════╗"
echo "║  Initial Setup Complete!                                  ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo
echo "Next steps:"
echo
echo "1. Configure Docker Compose environment:"
echo "   cd $REPO_DIR/compose"
echo "   cp .env.example .env"
echo "   nano .env"
echo
echo "2. Start services:"
echo "   cd $REPO_DIR/compose"
echo "   docker compose -f media.yml -f web.yml -f cloud.yml up -d"
echo
echo "3. Configure VPS reverse proxy (Nginx Proxy Manager)"
echo "   Point domains to: http://10.99.0.2:<port>"
echo
echo "4. Set up monitoring (optional):"
echo "   $REPO_DIR/scripts/nfs-health.sh"
echo "   $REPO_DIR/scripts/wireguard-check.sh"
echo
echo "5. Test GPU transcoding:"
echo "   $REPO_DIR/scripts/gpu-verify.sh"
echo
