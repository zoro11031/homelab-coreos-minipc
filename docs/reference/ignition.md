# CoreOS Ignition Configuration

This directory contains the Ignition configuration files needed for CoreOS installation. Ignition is the provisioning system used by Fedora CoreOS (and uCore) to configure machines on first boot.

## Overview

- **Butane** (`.bu`) - Human-readable YAML configuration format
- **Ignition** (`.ign`) - Machine-readable JSON format that CoreOS actually uses

You write Butane files, then transpile them to Ignition JSON for installation.

## Quick Start

### 1. Prepare Your Configuration

Copy the template and customize it:

```bash
cd ignition
cp config.bu.template config.bu
```

### 2. Generate Password Hash

Run the helper script to generate a password hash for the `core` user:

```bash
./generate-password-hash.sh
```

This will output a hash like:
```
$y$j9T$abc123...xyz789
```

Copy this hash and replace `YOUR_GOOD_PASSWORD_HASH_HERE` in `config.bu`.

### 3. Add Your SSH Public Key

Edit `config.bu` and replace the example SSH key with your actual public key:

```yaml
ssh_authorized_keys:
  - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... your-email@example.com"
```

You can find your SSH public key:
- Linux/macOS: `cat ~/.ssh/id_ed25519.pub` or `cat ~/.ssh/id_rsa.pub`
- Windows: `type %USERPROFILE%\.ssh\id_ed25519.pub`

If you don't have an SSH key, generate one:
```bash
ssh-keygen -t ed25519 -C "your-email@example.com"
```

### 4. Transpile to Ignition JSON

Convert your Butane file to Ignition JSON:

```bash
./transpile.sh config.bu config.ign
```

This creates `config.ign`, which you'll use during CoreOS installation.

### 5. Use the Ignition File

There are several ways to use the Ignition file:

#### Option A: Embed in ISO (Recommended for USB installation)

```bash
# Download the CoreOS ISO or use your custom BlueBuild ISO
# Then embed the Ignition config
coreos-installer iso ignition embed -i config.ign homelab-coreos-minipc.iso
```

Now the ISO will automatically configure the system on first boot.

#### Option B: Provide URL During Installation

Host the `config.ign` file on a web server and provide the URL during installation:

```bash
# During interactive installation
sudo coreos-installer install /dev/sda --ignition-url https://example.com/config.ign

# Or use a local file
sudo coreos-installer install /dev/sda --ignition-file config.ign
```

#### Option C: Use with VM Platforms

For virtualization platforms (libvirt, QEMU, etc.), provide the Ignition file as a config drive or through the platform's metadata service.

#### Option D: Use USB Drive During Installation (Physical Access)

If you have physical access to the machine, you can use a USB drive to provide the Ignition config during installation. This is particularly useful for bare-metal installations where you can't easily embed the config in the ISO.

##### Step 1: Identify Your Drives

First, boot from the Fedora CoreOS live ISO and identify your drives:

```bash
# List all block devices
lsblk

# Get detailed block device information including UUIDs
sudo blkid
```

Example output from `lsblk`:
```
NAME   MAJ:MIN RM   SIZE RO TYPE MOUNTPOINTS
sda      8:0    0 238.5G  0 disk          # Your SSD (install target)
sdb      8:16   1  14.9G  0 disk          # Your USB drive
├─sdb1   8:17   1  14.9G  0 part
```

Example output from `blkid`:
```
/dev/sda: TYPE="disk"
/dev/sdb1: UUID="ABCD-1234" LABEL="IGNITION" TYPE="vfat"
```

**Important**:
- `/dev/sda` (or similar) is typically your SSD where you'll install CoreOS
- `/dev/sdb` (or similar) is typically your USB drive
- **Double-check** before proceeding - installing to the wrong drive will erase it!

##### Step 2: Prepare Your USB Drive

If your USB drive doesn't already have a filesystem, format it:

```bash
# WARNING: This will erase all data on the USB drive!
# Replace /dev/sdb with your actual USB device

# Create a new partition table
sudo parted /dev/sdb mklabel gpt

# Create a single partition
sudo parted /dev/sdb mkpart primary fat32 1MiB 100%

# Format as FAT32 (widely compatible)
sudo mkfs.vfat -n IGNITION /dev/sdb1

# Verify
sudo blkid /dev/sdb1
```

##### Step 3: Mount the USB Drive

```bash
# Create a mount point
sudo mkdir -p /mnt/usb

# Mount the USB drive
sudo mount /dev/sdb1 /mnt/usb

# Verify it's mounted
mount | grep /mnt/usb
df -h /mnt/usb
```

##### Step 4: Copy Ignition Config to USB

```bash
# Copy your Ignition config to the USB drive
sudo cp config.ign /mnt/usb/

# Verify the copy
ls -lh /mnt/usb/config.ign

# Optional: Set permissions to be readable
sudo chmod 644 /mnt/usb/config.ign

# Sync to ensure data is written
sudo sync

# Unmount the USB drive
sudo umount /mnt/usb
```

##### Step 5: Install CoreOS with USB-Provided Config

Now you can install CoreOS to your SSD, using the Ignition config from the USB drive:

```bash
# Remount the USB drive (if you unplugged it)
sudo mount /dev/sdb1 /mnt/usb

# Install CoreOS to the SSD with Ignition config from USB
# Replace /dev/sda with your actual SSD device
sudo coreos-installer install /dev/sda \
  --ignition-file /mnt/usb/config.ign

# Alternative: If the USB is still mounted at /mnt/usb
sudo coreos-installer install /dev/sda -i /mnt/usb/config.ign
```

##### Step 6: Post-Installation

```bash
# Unmount the USB drive
sudo umount /mnt/usb

# Reboot into your new system
sudo systemctl reboot
```

**After reboot**: Remove the installation media and USB drive. The system will boot from the SSD with your Ignition configuration applied.

##### Troubleshooting USB Installation

**USB drive not detected:**
```bash
# Check if the kernel sees the device
dmesg | grep -i usb
dmesg | grep sd

# List all SCSI/SATA devices
ls -l /dev/sd*

# Check USB subsystem
lsusb
```

**Wrong device identifier:**
```bash
# Use multiple methods to confirm
lsblk -o NAME,SIZE,TYPE,MOUNTPOINT,MODEL
sudo fdisk -l
sudo parted -l
```

**Can't mount USB:**
```bash
# Check filesystem type
sudo blkid /dev/sdb1

# Try different filesystem types
sudo mount -t vfat /dev/sdb1 /mnt/usb
sudo mount -t ext4 /dev/sdb1 /mnt/usb

# Check for errors
dmesg | tail -20
```

**Ignition config not found:**
```bash
# Verify file exists on USB
sudo mount /dev/sdb1 /mnt/usb
ls -la /mnt/usb/
cat /mnt/usb/config.ign | head -20

# Check file permissions
sudo chmod 644 /mnt/usb/config.ign
```

**Installation fails:**
```bash
# Verify the target disk is correct and unmounted
sudo umount /dev/sda* 2>/dev/null
lsblk /dev/sda

# Check coreos-installer version
coreos-installer --version

# Try with verbose output
sudo coreos-installer install /dev/sda \
  --ignition-file /mnt/usb/config.ign \
  -v
```

##### Tips for Physical Installation

1. **Label your USB drive** - Use `IGNITION` as the label for easy identification
2. **Keep backups** - Save `config.ign` to multiple locations
3. **Test in VM first** - Validate your Ignition config in a VM before physical installation
4. **Document your setup** - Note which device is which (take a photo of `lsblk` output)
5. **Use stable device names** - When possible, use `/dev/disk/by-id/*` for more stable device identification:
   ```bash
   ls -l /dev/disk/by-id/
   # Use the full path like:
   # /dev/disk/by-id/ata-Samsung_SSD_870_EVO_500GB_S1234567890
   ```

## What the Configuration Does

The included `config.bu.template` configures:

1. **User Setup**: Sets password and SSH key for the `core` user
2. **Groups**: Adds `core` to `wheel` and `docker` groups for elevated privileges
3. **Hostname**: Sets the system hostname to `homelab-minipc`
4. **Automatic Rebase**: Sets up systemd services for automatic rebasing to the custom image

### Automatic Rebase Workflow

The configuration includes two systemd services that handle automatic rebasing to your custom `homelab-coreos-minipc` image:

**First Boot (Unsigned Rebase)**
- The `ucore-unsigned-autorebase.service` runs on first boot
- Rebases the system to `ostree-unverified-registry:ghcr.io/plex-migration-homelab/homelab-coreos-minipc:latest`
- Creates a marker file at `/etc/ucore-autorebase/unverified`
- Disables itself and reboots

**Second Boot (Signed Rebase)**
- The `ucore-signed-autorebase.service` runs after the first reboot
- Rebases the system to the signed image `ostree-image-signed:docker://ghcr.io/plex-migration-homelab/homelab-coreos-minipc:latest`
- Creates a marker file at `/etc/ucore-autorebase/signed`
- Disables itself and reboots

After these two automatic reboots, your system will be running the fully signed custom image and ready for use.

**Important**: This means your system will reboot **twice automatically** during initial setup. This is normal and expected behavior.

### Disabling the Automatic Rebase Units

If you generate an ISO directly from the published image (e.g. `sudo bluebuild generate-iso --iso-name homelab-coreos-minipc.iso image ghcr.io/plex-migration-homelab/homelab-coreos-minipc`) and embed this Ignition file before flashing, the machine already boots into the intended image on first launch. In that case the autorebase services are redundant and will only introduce two extra reboots. Remove them before running `transpile.sh`:

1. Delete the `storage.directories` entry that creates `/etc/ucore-autorebase`.
2. Remove both `ucore-unsigned-autorebase.service` and `ucore-signed-autorebase.service` from the `systemd.units` list.

With those sections removed the Ignition file will leave the system as-is, avoiding unnecessary reboots while keeping the rest of your configuration intact.

## Advanced Configuration

You can extend the Butane file to include:

- **Additional users**
- **System files** (WireGuard configs, systemd units, etc.)
- **Systemd services** to enable/disable
- **Network configuration**
- **Disk partitioning**

See the [Butane documentation](https://coreos.github.io/butane/) for more options.

## Example: Adding a Systemd Unit

```yaml
systemd:
  units:
    - name: docker.service
      enabled: true
    - name: wg-quick@wg0.service
      enabled: true
```

## Example: Adding a Configuration File

```yaml
storage:
  files:
    - path: /etc/wireguard/wg0.conf
      mode: 0600
      contents:
        inline: |
          [Interface]
          PrivateKey = your-private-key
          Address = 10.253.0.1/24
          ListenPort = 51820
```

## Security Notes

- **Never commit `config.bu` or `config.ign` to Git** if they contain real passwords or keys
- The `.gitignore` in this repo already excludes `*.bu` and `*.ign` files (except templates)
- Store sensitive Ignition configs securely (password manager, encrypted storage, etc.)
- Use SSH keys instead of passwords when possible
- Consider using temporary Ignition configs and provisioning secrets after installation

## Troubleshooting

### Can't find mkpasswd

Install it based on your OS:
- Fedora/RHEL: `sudo dnf install mkpasswd`
- Debian/Ubuntu: `sudo apt install whois`
- macOS: `brew install mkpasswd`

### Can't find butane

Download from [GitHub releases](https://github.com/coreos/butane/releases) or:
- Fedora: `sudo dnf install butane`
- Manual: Download the binary and place it in `/usr/local/bin/`

### Ignition validation errors

Run butane with `--strict` flag (already included in `transpile.sh`) to catch errors early:

```bash
butane --pretty --strict < config.bu
```

### SSH key not working

Make sure:
1. The SSH key format is correct (starts with `ssh-ed25519`, `ssh-rsa`, etc.)
2. The entire key is on one line in the Butane file
3. There are no extra quotes or spaces
4. You're using the corresponding private key when connecting

## Additional Resources

- [Fedora CoreOS Documentation](https://docs.fedoraproject.org/en-US/fedora-coreos/)
- [Butane Configuration Specification](https://coreos.github.io/butane/config-fcos-v1_5/)
- [Ignition Documentation](https://coreos.github.io/ignition/)
