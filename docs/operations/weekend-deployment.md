# Weekend Server Deployment Guide

## ✅ Pre-Deployment Checklist - COMPLETE

All systems are **GO** for weekend deployment! ✓

### Binary Status
- ✓ Binary built and tested: `7.1M`
- ✓ Location in image: `files/system/usr/local/bin/homelab-setup`
- ✓ Will deploy to: `/usr/local/bin/homelab-setup`
- ✓ All commands verified working
- ✓ Git tracked and ready to commit
- ✓ Architecture compatible: x86-64 (Fedora CoreOS)

### CI/CD Status
- ✓ GitHub Actions workflow configured
- ✓ Auto-rebuild on Go code changes
- ✓ BlueBuild integration ready

## 🚀 Deployment Steps

### 1. Commit and Push Changes
```bash
git add .
git commit -m "feat: add homelab-setup Go binary and auto-build workflow"
git push origin main
```

### 2. Wait for Image Build
- GitHub Actions will trigger BlueBuild
- Check: https://github.com/zoro11031/homelab-coreos-minipc/actions
- Wait for green checkmark (build complete)

### 3. Deploy on Server

#### Option A: Rebase to new image (Recommended)
```bash
# SSH to your server
ssh your-server

# Rebase to the new image
sudo rpm-ostree rebase ostree-image-signed:docker://ghcr.io/zoro11031/homelab-coreos-minipc:latest

# Reboot to apply
sudo systemctl reboot
```

#### Option B: Update existing system
```bash
# SSH to your server
ssh your-server

# Update to latest image
sudo rpm-ostree upgrade

# Reboot to apply
sudo systemctl reboot
```

### 4. Verify Installation

After reboot, SSH back in and verify:

```bash
# Check binary is present
which homelab-setup
# Should output: /usr/local/bin/homelab-setup

# Check version
homelab-setup version
# Should show version info with build date

# Check it's working
homelab-setup status
# Should show setup status (0/7 steps)
```

### 5. Run Setup

```bash
# Start the interactive setup
homelab-setup

# Or run individual steps
homelab-setup run preflight
homelab-setup run user
# ... etc
```

## 🔍 Verification Commands

```bash
# Check setup status
homelab-setup status

# Run diagnostics
homelab-setup troubleshoot

# View help
homelab-setup --help

# Reset if needed
homelab-setup reset
```

## ⚠️ Server Requirements

Make sure your server has:

1. **Passwordless sudo** - Required for system operations
   ```bash
   # Check with:
   sudo -n true && echo "✓ Configured" || echo "✗ Not configured"

   # Configure if needed:
   echo "$USER ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/$USER
   sudo chmod 0440 /etc/sudoers.d/$USER
   ```

2. **UBlue uCore** - Fedora CoreOS based system with rpm-ostree

3. **Network access** - For pulling images and running setup

## 🐛 Troubleshooting

### Binary not found after reboot
```bash
# Check if binary exists
ls -la /usr/local/bin/homelab-setup

# If missing, check image was applied
rpm-ostree status

# Verify you're on the right image
rpm-ostree status | grep homelab-coreos-minipc
```

### Permission denied
```bash
# Check binary permissions
ls -la /usr/local/bin/homelab-setup
# Should be: -rwxr-xr-x root:root

# If wrong, the image needs to be rebuilt
```

### Command not working
```bash
# Run troubleshoot to diagnose
homelab-setup troubleshoot

# Check logs
journalctl -xe | grep homelab

# Verify sudo works
sudo -n true
```

## 📁 File Locations

After deployment:
- Binary: `/usr/local/bin/homelab-setup`
- Config: `~/.homelab-setup.conf`
- Markers: `~/.local/homelab-setup/`
- Logs: System journal (use `journalctl`)

## 🎯 Quick Start After Deployment

```bash
# 1. Verify binary
homelab-setup version

# 2. Check status
homelab-setup status

# 3. Run setup
homelab-setup

# 4. Follow the interactive prompts
```

## 🔄 Future Updates

When you update the Go code in `homelab-setup/`:
1. Push changes to GitHub
2. GitHub Actions automatically rebuilds binary
3. Binary is committed back to repo
4. Next BlueBuild includes updated binary
5. Update server with `rpm-ostree upgrade` and reboot

---

**Ready to deploy!** 🎉

All tests passed. The binary is production-ready and will work on your server.
