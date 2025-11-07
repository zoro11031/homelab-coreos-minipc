# Virt-Manager QA Guide for `homelab-coreos-minipc`

This guide describes how to validate the `homelab-coreos-minipc` uCore image end to end inside a KVM/libvirt environment with **virt-manager**. Follow these steps before promoting a new image so that storage mounts, WireGuard connectivity, Podman Compose workloads, and migration workflows are all proven to work.

## 1. Test Environment Preparation

1. **Host requirements**
   - Workstation with hardware virtualization (VT-x/AMD-V) enabled.
   - Fedora/RHEL, Debian, or Arch-based distro with `libvirt`, `virt-manager`, and `virt-install` packages installed.
   - Enough CPU (4 vCPU), RAM (12–16 GiB), and disk (80 GiB) to mirror the NAB9 mini PC footprint.
2. **Download the image under test**
   - Pull the latest build from GitHub Container Registry: `podman pull ghcr.io/<user>/homelab-coreos-minipc:latest`.
   - Use `podman image save --format oci-archive` or `skopeo` to export the root filesystem for CoreOS live ISO use, then write it to a qcow2 disk for testing.
3. **Provision an ignition file**
   - Start with `ignition/minipc.ign` (or the generated output) and adjust credentials as needed for the lab VM.
   - Keep the default `core` user so that systemd setup scripts seeded by the build can run. They stage dotfiles, Compose bundles, and WireGuard helpers under `/home/core/setup` via the `home-directory-setup.service` unit.【F:files/system/etc/systemd/system/home-directory-setup.service†L1-L15】【F:files/system/usr/share/setup-scripts/compose.sh†L1-L5】

## 2. Create the VM in virt-manager

1. Launch **virt-manager** and create a **New Virtual Machine**.
2. Choose **Import existing disk image**, point to the qcow2 file, and select **Fedora CoreOS** as the OS type.
3. Allocate **4 vCPU**, **12 GiB RAM**, and set firmware to **UEFI**.
4. Configure **Networking**
   - Attach one NIC to a libvirt bridge that provides outbound internet.
   - Optionally add a second NIC on an isolated network to emulate LAN-only services.
5. Add **virtiofs-backed storage shares** for NFS validation
   - Use **Add Hardware → Filesystem → virtiofs** to expose host directories that simulate the NAS exports expected by the image.
   - Map host paths to guest mount points `/mnt/nas-media`, `/mnt/nas-nextcloud`, and `/mnt/nas-immich` to satisfy the systemd mount units that ship with the build.【F:files/system/etc/systemd/system/mnt-nas-media.mount†L1-L19】【F:files/system/etc/systemd/system/mnt-nas-nextcloud.mount†L1-L19】【F:files/system/etc/systemd/system/mnt-nas-immich.mount†L1-L19】
6. Attach any additional virtual disks needed for migration rehearsal (e.g., a second qcow2 for exporting container data).

## 3. First Boot Verification

1. Boot the VM and watch the serial console for successful ignition application.
2. Log in as `core` via the console.
3. Confirm that the setup service has run:
   ```bash
   systemctl status home-directory-setup.service
   ls -R /home/core/setup
   ```
4. Validate that the Compose bundles and WireGuard helper scripts are present under `/home/core/setup` and owned by the `core` user.【F:files/system/usr/share/setup-scripts/wireguard.sh†L1-L4】【F:files/system/usr/share/setup-scripts/dotfiles.sh†L1-L6】【F:files/system/usr/share/setup-scripts/compose.sh†L1-L5】
5. Capture the VM state (snapshot) now so you can roll back during troubleshooting.

## 4. Storage Mount QA inside the VM

1. **Mount unit health**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart mnt-nas-media.mount mnt-nas-nextcloud.mount mnt-nas-immich.mount
   systemctl status mnt-nas-media.mount mnt-nas-nextcloud.mount mnt-nas-immich.mount
   ```
2. **Mountpoint verification**
   ```bash
   findmnt /mnt/nas-media
   findmnt /mnt/nas-nextcloud
   findmnt /mnt/nas-immich
   ```
3. **Read/write validation**
   - For read-only media share (`/mnt/nas-media`), check permissions: `touch /mnt/nas-media/test` should fail with `Read-only file system`.
   - For read/write shares, create and delete sentinel files: `sudo touch /mnt/nas-nextcloud/.qa` then `sudo rm /mnt/nas-nextcloud/.qa`.
4. **Hot-detach scenarios**
   - In virt-manager, remove a virtiofs share and observe `systemctl status` transitions (should retry because of `_netdev` and `TimeoutSec=60`).【F:files/system/etc/systemd/system/mnt-nas-media.mount†L8-L19】
   - Reattach the share and ensure `systemctl restart` recovers the mount.

## 5. WireGuard Configuration Testing

1. Enter the WireGuard staging directory: `cd /home/core/setup/wireguard-setup`.
2. Generate fresh lab keys: `sudo ./generate-keys.sh`. This script scaffolds `.env`, individual key files, and guides the next steps.【F:files/system/usr/share/wireguard-setup/generate-keys.sh†L1-L143】
3. Run `sudo ./apply-config.sh` to materialize `wg0.conf` from the template, ensuring placeholders resolve using the generated environment variables.【F:files/system/usr/share/wireguard-setup/apply-config.sh†L1-L115】
4. Copy `wg0.conf` into `/etc/wireguard/`, enable the service, and verify status:
   ```bash
   sudo cp wg0.conf /etc/wireguard/wg0.conf
   sudo systemctl enable --now wg-quick@wg0
   sudo systemctl status wg-quick@wg0
   sudo wg show
   ```
   The custom `wg-quick@.service` unit ensures restarts on failure and hooks into the standard target graph.【F:files/system/etc/systemd/system/wg-quick@.service†L1-L22】
5. Exercise connectivity by spinning up a WireGuard peer on your workstation using the generated keys and pinging `10.253.0.1`.
6. Simulate endpoint loss by disconnecting the VM NIC; confirm that `wg-quick@wg0` automatically retries because of `Restart=on-failure`.

## 6. Podman Compose Stack Validation

1. Stage the compose workspace:
   ```bash
   sudo cp -r /home/core/setup/compose-setup /var/home/core/compose
   sudo chown -R core:core /var/home/core/compose
   cd /var/home/core/compose
   ```
2. Create an `.env` file with the secrets referenced by the bundles (see `cloud.yml`, `media.yml`, and `web.yml`).【F:files/system/usr/share/compose-setup/cloud.yml†L1-L93】【F:files/system/usr/share/compose-setup/media.yml†L1-L88】【F:files/system/usr/share/compose-setup/web.yml†L1-L84】
3. Start each stack individually to isolate failures:
   ```bash
   podman compose -f media.yml up -d
   podman compose -f web.yml up -d
   podman compose -f cloud.yml up -d
   ```
4. Inspect container health and logs:
   ```bash
   podman ps --format '{{.Names}}\t{{.Status}}'
   podman logs nextcloud
   podman logs jellyfin
   ```
5. Confirm that services relying on NFS succeed (Nextcloud DB volume, Immich libraries). Missing mounts will surface as permission errors; correlate with Section 4.
6. Tear down and rebuild to ensure idempotence: `podman compose -f media.yml down --volumes` followed by another `up` cycle.

## 7. Migration Rehearsal

1. **Simulate state backup**
   - Attach an extra qcow2 disk in virt-manager (virtio) to represent the production boot disk.
   - Use `podman volume export` and `tar` to capture application data into that disk.
2. **rpm-ostree upgrade rehearsal**
   - Trigger a rebase to the image under test: `sudo rpm-ostree rebase ostree-unverified-registry:ghcr.io/<user>/homelab-coreos-minipc:latest` (matches the recipe base in `recipes/recipe.yml`).【F:recipes/recipe.yml†L1-L17】
   - Reboot and confirm `rpm-ostree status` shows the new deployment.
3. **Rollback drill**
   - Run `sudo rpm-ostree rollback` to verify that the previous deployment remains available and stable.
4. **Data restore**
   - Re-import saved volumes and rerun `podman compose up` to prove the stack survives migrations without data loss.

## 8. Troubleshooting Checklist

| Symptom | Likely Cause | Mitigation |
|---------|--------------|------------|
| Mount units stuck in `activating` | virtiofs share missing or wrong path | Reattach share, rerun `systemctl restart mnt-nas-*.mount`. |
| `wg-quick@wg0` fails at boot | `.env` missing keys or NIC mismatch | Regenerate keys and set `WG_OUTBOUND_INTERFACE` before rerunning `apply-config.sh`.【F:files/system/usr/share/wireguard-setup/apply-config.sh†L31-L82】 |
| Compose services exit immediately | Secrets/env vars missing | Populate `.env` with all variables referenced in the compose YAML files.【F:files/system/usr/share/compose-setup/cloud.yml†L9-L73】 |
| Nextcloud volume permissions errors | Virtiofs share exported read-only | Re-export with read/write or adjust `Options` on the share to mirror NFS expectations.【F:files/system/etc/systemd/system/mnt-nas-nextcloud.mount†L10-L16】 |
| rpm-ostree rebase blocked | Image registry unreachable | Verify host networking and that the target tag exists in GHCR. |

## 9. Exit Criteria

A build is considered QA-complete when:

- Storage mounts pass attach/detach and read/write checks.
- WireGuard tunnel forms, survives restarts, and peers can exchange traffic.
- Podman Compose workloads deploy cleanly and restart without residual errors.
- Migration rehearsal (backup → rebase → rollback) executes without data loss or boot failures.

Document the results in the release checklist and attach console logs or screenshots for any anomalies discovered during virt-manager testing.
