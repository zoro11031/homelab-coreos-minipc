# Troubleshooting Guide

Common issues and solutions for the NAB9 mini PC homelab setup.

## Table of Contents

1. [Boot and System Issues](#boot-and-system-issues)
2. [Network Issues](#network-issues)
3. [NFS Mount Issues](#nfs-mount-issues)
4. [WireGuard VPN Issues](#wireguard-vpn-issues)
5. [Docker and Container Issues](#docker-and-container-issues)
6. [Service-Specific Issues](#service-specific-issues)
7. [Hardware and GPU Issues](#hardware-and-gpu-issues)
8. [Performance Issues](#performance-issues)

## Boot and System Issues

### System Won't Boot

**Symptoms:** Black screen, no POST, no boot

**Solutions:**
1. Check power connection and LED indicators
2. Connect via IP KVM to see BIOS screen
3. Try different display output if using monitor
4. Reset BIOS to defaults (CMOS clear)
5. Check boot order in BIOS (NVMe should be first)

### System Boots to Emergency Mode

**Symptoms:** "Emergency mode" or "You are in rescue mode"

**Solutions:**
1. This usually indicates failed mounts
2. Check NFS server is online and accessible
3. Disable NFS mounts temporarily:
   ```bash
   # From rescue shell
   systemctl mask mnt-nas-media.mount
   systemctl mask mnt-nas-nextcloud.mount
   systemctl mask mnt-nas-immich.mount
   systemctl reboot
   ```
4. After boot, investigate and fix NFS issues
5. Re-enable mounts:
   ```bash
   systemctl unmask mnt-nas-media.mount
   systemctl start mnt-nas-media.mount
   ```

### Can't SSH Into System

**Symptoms:** "Connection refused" or timeout

**Solutions:**
1. Verify network cable is connected
2. Check IP address via IP KVM or monitor
3. Ping the system: `ping 192.168.7.x`
4. Check if SSH service is running (via KVM):
   ```bash
   systemctl status sshd
   ```
5. Check firewall:
   ```bash
   sudo ufw status
   sudo ufw allow 22/tcp
   ```
6. Verify SSH keys are configured correctly

### System Slow After Boot

**Symptoms:** High load, slow response

**Solutions:**
1. Check what's consuming resources:
   ```bash
   top
   htop
   systemctl list-units --type=service --state=running
   ```
2. Check for failed mounts:
   ```bash
   systemctl --failed
   ```
3. Check disk I/O:
   ```bash
   iotop
   ```
4. Look for Docker container issues:
   ```bash
   docker stats
   ```

## Network Issues

### No Network Connectivity

**Symptoms:** Can't reach internet or local network

**Solutions:**
1. Check physical connection:
   ```bash
   ip link show
   # Look for "UP" status on ethernet interface
   ```
2. Check IP configuration:
   ```bash
   ip addr show
   ```
3. Restart networking:
   ```bash
   sudo systemctl restart NetworkManager
   ```
4. Check routes:
   ```bash
   ip route show
   ```
5. Test gateway:
   ```bash
   ping 192.168.7.1  # Your router
   ```

### DNS Resolution Failing

**Symptoms:** Can ping IPs but not domain names

**Solutions:**
1. Check DNS configuration:
   ```bash
   cat /etc/resolv.conf
   ```
2. Test DNS:
   ```bash
   nslookup google.com
   dig google.com
   ```
3. Try different DNS server:
   ```bash
   sudo nano /etc/resolv.conf
   # Add: nameserver 1.1.1.1
   ```
4. Restart NetworkManager:
   ```bash
   sudo systemctl restart NetworkManager
   ```

### Firewall Blocking Traffic

**Symptoms:** Some services unreachable

**Solutions:**
1. Check UFW status:
   ```bash
   sudo ufw status verbose
   ```
2. Check recent blocks:
   ```bash
   sudo tail -f /var/log/ufw.log
   ```
3. Temporarily disable to test:
   ```bash
   sudo ufw disable
   # Test connectivity
   sudo ufw enable
   ```
4. Add rules as needed:
   ```bash
   sudo ufw allow from 192.168.7.0/24
   sudo ufw allow 32400/tcp
   ```

## NFS Mount Issues

### NFS Mounts Not Working

**Symptoms:** Mount points empty, services can't access media

**Solutions:**
1. Check mount status:
   ```bash
   systemctl status mnt-nas-media.mount
   systemctl status mnt-nas-nextcloud.mount
   systemctl status mnt-nas-immich.mount
   ```
2. Check if NFS server is accessible:
   ```bash
   ping 192.168.7.10
   showmount -e 192.168.7.10
   ```
3. Try manual mount:
   ```bash
   sudo mount -t nfs 192.168.7.10:/mnt/storage/Media /mnt/nas-media
   ```
4. Check NFS client packages:
   ```bash
   rpm -qa | grep nfs-utils
   ```
5. Check server exports (on file server):
   ```bash
   cat /etc/exports
   exportfs -v
   ```

### NFS Mount Hangs

**Symptoms:** System hangs when accessing mount point

**Solutions:**
1. Check NFS server is responding:
   ```bash
   ping 192.168.7.10
   ```
2. Force unmount:
   ```bash
   sudo umount -f /mnt/nas-media
   ```
3. Check for stale handles:
   ```bash
   sudo systemctl restart nfs-client.target
   ```
4. Modify mount options to be more resilient:
   ```bash
   # Edit mount unit
   sudo nano /etc/systemd/system/mnt-nas-media.mount
   # Change Options= to include:
   # soft,intr,timeo=10,retrans=1
   ```

### Permission Denied on NFS Mount

**Symptoms:** Can see files but can't read/write

**Solutions:**
1. Check file permissions:
   ```bash
   ls -la /mnt/nas-media
   ```
2. Verify UID/GID match between systems:
   ```bash
   id  # On mini PC
   # Should match file ownership on NFS server
   ```
3. Check NFS export options on server:
   ```bash
   # On file server
   cat /etc/exports
   # Should have: rw,sync,no_subtree_check,no_root_squash
   ```
4. Re-export on server:
   ```bash
   # On file server
   sudo exportfs -ra
   ```

### Use Health Check Script

```bash
cd ~/homelab-coreos-minipc
./scripts/nfs-health.sh
```

This automatically checks and attempts to remount failed NFS shares.

## WireGuard VPN Issues

### WireGuard Not Connecting

**Symptoms:** Can't reach VPS, services not accessible remotely

**Solutions:**
1. Check service status:
   ```bash
   sudo systemctl status wg-quick@wg0
   ```
2. Check interface:
   ```bash
   sudo wg show
   ip addr show wg0
   ```
3. Test connectivity to VPS:
   ```bash
   ping YOUR_VPS_IP
   ping 10.99.0.1  # VPS internal IP
   ```
4. Check configuration:
   ```bash
   sudo wg show wg0
   ```
5. Restart WireGuard:
   ```bash
   sudo systemctl restart wg-quick@wg0
   ```

### WireGuard Connected But No Traffic

**Symptoms:** Interface up but can't reach services

**Solutions:**
1. Check allowed IPs in config:
   ```bash
   sudo cat /etc/wireguard/wg0.conf | grep AllowedIPs
   ```
2. Check routing:
   ```bash
   ip route show table all | grep wg0
   ```
3. Check firewall on VPS
4. Test from VPS to mini PC:
   ```bash
   # On VPS
   ping 10.99.0.2
   curl http://10.99.0.2:5055  # Overseerr
   ```

### WireGuard Keys Not Working

**Symptoms:** "Bad message" or authentication errors

**Solutions:**
1. Verify keys are correct (no extra whitespace)
2. Regenerate keys:
   ```bash
   wg genkey | tee privatekey | wg pubkey > publickey
   ```
3. Update both client and server configs
4. Restart both ends:
   ```bash
   # On mini PC
   sudo systemctl restart wg-quick@wg0

   # On VPS
   sudo systemctl restart wg-quick@wg0
   ```

### Use Health Check Script

```bash
cd ~/homelab-coreos-minipc
./scripts/wireguard-check.sh
```

This automatically checks and attempts to reconnect WireGuard.

## Docker and Container Issues

### Docker Service Won't Start

**Symptoms:** "docker: command not found" or service failed

**Solutions:**
1. Check if Docker is installed:
   ```bash
   rpm -qa | grep docker
   ```
2. Check service status:
   ```bash
   sudo systemctl status docker
   ```
3. Check logs:
   ```bash
   sudo journalctl -u docker -n 50
   ```
4. Start Docker:
   ```bash
   sudo systemctl start docker
   sudo systemctl enable docker
   ```

### Container Won't Start

**Symptoms:** Container exits immediately or shows error

**Solutions:**
1. Check container status:
   ```bash
   docker ps -a
   ```
2. View container logs:
   ```bash
   docker logs container_name
   ```
3. Check for port conflicts:
   ```bash
   sudo netstat -tlnp | grep PORT
   ```
4. Try starting manually:
   ```bash
   docker start container_name
   docker logs -f container_name
   ```
5. Recreate container:
   ```bash
   cd ~/homelab-coreos-minipc/compose
   docker compose -f media.yml up -d --force-recreate plex
   ```

### Container Can't Access NFS Mounts

**Symptoms:** "No such file or directory" in container

**Solutions:**
1. Verify mounts are active:
   ```bash
   df -h | grep nfs
   ```
2. Check bind mount in compose file:
   ```yaml
   volumes:
     - /mnt/nas-media:/media:ro
   ```
3. Restart Docker (after mounts are up):
   ```bash
   sudo systemctl restart docker
   ```
4. Recreate containers:
   ```bash
   docker compose -f media.yml up -d --force-recreate
   ```

### Out of Disk Space

**Symptoms:** "No space left on device"

**Solutions:**
1. Check disk usage:
   ```bash
   df -h
   du -sh /var/lib/containers/*
   ```
2. Clean up Docker:
   ```bash
   docker system prune -a
   docker volume prune
   ```
3. Check transcode temp directory:
   ```bash
   du -sh /var/lib/containers/appdata/plex/transcode
   rm -rf /var/lib/containers/appdata/plex/transcode/*
   ```
4. Check logs:
   ```bash
   sudo journalctl --vacuum-time=7d
   ```

## Service-Specific Issues

### Plex

#### Can't Access Plex Remotely

**Solutions:**
1. Check port forwarding on router (port 32400)
2. Test direct access: http://YOUR_PUBLIC_IP:32400/web
3. Check Plex remote access settings
4. Verify firewall allows 32400:
   ```bash
   sudo ufw allow 32400/tcp
   ```

#### Plex Not Transcoding

**Solutions:**
1. Check GPU configuration (see GPU_TRANSCODING.md)
2. Verify Plex Pass subscription
3. Check transcoder settings
4. Monitor during playback:
   ```bash
   docker logs -f plex
   sudo intel_gpu_top
   ```

### Jellyfin

#### Jellyfin Won't Stream

**Solutions:**
1. Check hardware acceleration settings
2. Verify media files are accessible:
   ```bash
   docker exec jellyfin ls /media
   ```
3. Check logs:
   ```bash
   docker logs jellyfin | grep -i error
   ```

### Overseerr

#### Can't Connect to Plex/Jellyfin

**Solutions:**
1. Use local IP addresses: http://192.168.7.x:PORT
2. Don't use localhost (won't work in container)
3. Verify Plex/Jellyfin are running:
   ```bash
   docker ps | grep -E "plex|jellyfin"
   ```

### Nextcloud

#### Can't Access Nextcloud

**Solutions:**
1. Check AIO master container:
   ```bash
   docker logs nextcloud-aio-mastercontainer
   ```
2. Access admin interface: http://192.168.7.x:8080
3. Check if Nextcloud containers are running:
   ```bash
   docker ps | grep nextcloud
   ```
4. Restart AIO:
   ```bash
   docker restart nextcloud-aio-mastercontainer
   ```

### Immich

#### Mobile App Can't Connect

**Solutions:**
1. Use full URL: http://192.168.7.x:2283
2. Don't use HTTPS for local access (unless configured)
3. Check if all Immich containers are running:
   ```bash
   docker ps | grep immich
   ```
4. Check logs:
   ```bash
   docker logs immich-server
   ```

#### Photos Not Uploading

**Solutions:**
1. Check NFS mount has write permissions:
   ```bash
   docker exec immich-server touch /usr/src/app/upload/test
   ```
2. Check disk space:
   ```bash
   df -h /mnt/nas-immich
   ```
3. Check container logs for errors

## Hardware and GPU Issues

See [GPU_TRANSCODING.md](GPU_TRANSCODING.md) for detailed GPU troubleshooting.

### Quick GPU Checks

```bash
# Verify GPU exists
lspci | grep VGA

# Check /dev/dri
ls -l /dev/dri/

# Test VA-API
vainfo

# Run verification script
~/homelab-coreos-minipc/scripts/gpu-verify.sh
```

## Performance Issues

### High CPU Usage

**Solutions:**
1. Identify process:
   ```bash
   top
   htop
   ```
2. Check if transcoding without GPU:
   ```bash
   sudo intel_gpu_top
   # GPU should show usage during transcodes
   ```
3. Check Docker containers:
   ```bash
   docker stats
   ```

### High Memory Usage

**Solutions:**
1. Check memory usage:
   ```bash
   free -h
   ```
2. Identify memory hogs:
   ```bash
   ps aux --sort=-%mem | head
   docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}"
   ```
3. Restart problem containers:
   ```bash
   docker restart container_name
   ```

### Slow NFS Performance

**Solutions:**
1. Test network speed:
   ```bash
   iperf3 -c 192.168.7.10
   ```
2. Check NFS mount options (should have large rsize/wsize):
   ```bash
   mount | grep nfs
   ```
3. Consider enabling jumbo frames:
   ```bash
   sudo ip link set eth0 mtu 9000
   ```

### Disk I/O Issues

**Solutions:**
1. Monitor I/O:
   ```bash
   iotop
   ```
2. Check for failing disk:
   ```bash
   sudo smartctl -a /dev/nvme0n1
   ```
3. Move transcode directory to faster storage

## Getting Help

### Collect Diagnostic Information

When seeking help, collect this information:

```bash
#!/bin/bash
# Diagnostic information collector

echo "=== System Info ==="
hostnamectl
uname -a

echo "=== Network ==="
ip addr
ip route

echo "=== Mounts ==="
df -h
mount | grep nfs

echo "=== Docker ==="
docker ps -a
docker stats --no-stream

echo "=== Services ==="
systemctl status wg-quick@wg0
systemctl status docker
systemctl status mnt-nas-media.mount

echo "=== GPU ==="
lspci | grep VGA
ls -l /dev/dri/
vainfo

echo "=== Logs ==="
sudo journalctl -xe -n 100
```

### Log Files

Important log locations:
- System: `journalctl -xe`
- Docker: `docker logs container_name`
- NFS: `/var/log/nfs-health.log`
- WireGuard: `/var/log/wireguard-health.log`
- Firewall: `/var/log/ufw.log`

### Online Resources

- [Ublue Discourse](https://universal-blue.discourse.group/)
- [Plex Forums](https://forums.plex.tv/)
- [Jellyfin Documentation](https://jellyfin.org/docs/)
- [WireGuard Documentation](https://www.wireguard.com/)
- [Docker Documentation](https://docs.docker.com/)
