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
- Rebases the system to `ostree-unverified-registry:ghcr.io/zoro11031/homelab-coreos-minipc:latest`
- Creates a marker file at `/etc/ucore-autorebase/unverified`
- Disables itself and reboots

**Second Boot (Signed Rebase)**
- The `ucore-signed-autorebase.service` runs after the first reboot
- Rebases the system to the signed image `ostree-image-signed:docker://ghcr.io/zoro11031/homelab-coreos-minipc:latest`
- Creates a marker file at `/etc/ucore-autorebase/signed`
- Disables itself and reboots

After these two automatic reboots, your system will be running the fully signed custom image and ready for use.

**Important**: This means your system will reboot **twice automatically** during initial setup. This is normal and expected behavior.

### Disabling the Automatic Rebase Units

If you generate an ISO directly from the published image (e.g. `sudo bluebuild generate-iso --iso-name homelab-coreos-minipc.iso image ghcr.io/zoro11031/homelab-coreos-minipc`) and embed this Ignition file before flashing, the machine already boots into the intended image on first launch. In that case the autorebase services are redundant and will only introduce two extra reboots. Remove them before running `transpile.sh`:

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
