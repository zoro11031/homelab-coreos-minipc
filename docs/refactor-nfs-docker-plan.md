# Refactor Directive: NFS Mounting & Docker-Based Compose Services

You are executing my orders, not freelancing. Follow this plan precisely. Deviations require my approval.

## Objectives
- Replace brittle systemd mount units with fstab-driven NFS mounts using resilient boot semantics.
- Regenerate compose-backed systemd services that depend on Docker and the NFS mount in the correct order.
- Add strict preflight validation (Docker, compose, mount availability) before deployment.
- Provide migration from legacy systemd mount/automount units to fstab.

## Ground Rules
- Use Go 1.23 patterns already present in `homelab-setup/`; keep changes small and readable.
- Tests are mandatory for new logic; use table-driven tests for helpers.
- No shell injection: only use trusted args when invoking commands.
- Preserve existing config keys; add new ones only if unavoidable.
- Keep user-facing messages terse and actionable.

## Work Packages

### 1) NFS setup moves to fstab (replace systemd mount units)
- Delete the old `createSystemdUnits` path in `internal/steps/nfs.go`.
- Implement `createFstabEntry(host, export, mountPoint)`:
  - Format: `{host}:{export} {mountPoint} nfs nfsvers=4.2,_netdev,nofail,defaults 0 0`.
  - Skip if an identical entry already exists (exact match on source + target).
  - After writing, run `mount -a --fake` to validate syntax, then `mount -a` and `findmnt {mountPoint}` to confirm the live mount.
  - Emit clear errors with remediation hints (check network, NFS server, permissions).
- Update `RunNFSSetup` to call the new helper, persist the mount point in config, and surface failures immediately.

### 2) Compose services depend on Docker and the NFS mount
- In `internal/steps/deployment.go`, add `fstabMountToSystemdUnit(mountPoint)` to escape paths (`systemd-escape -p --suffix=mount`).
- Enhance `createComposeService`:
  - `[Unit]`: include `Wants=network-online.target docker.service`, `After=network-online.target docker.service`, plus `After/Requires={escaped}.mount` and `RequiresMountsFor={mountPoint}` when NFS is configured.
  - `[Service]`: `Type=oneshot`, `RemainAfterExit=yes`, `WorkingDirectory={serviceDir}`.
    - `ExecStartPre=/usr/bin/findmnt {mountPoint}` when applicable.
    - `ExecStartPre={composeCmd} pull --quiet`.
    - `ExecStart={composeCmd} up -d --remove-orphans`; `ExecStop={composeCmd} down --timeout 30`.
    - `Restart=on-failure`, `RestartSec=10`, `TimeoutStartSec=600`, `TimeoutStopSec=120`.
  - Compose command detection: prefer `docker compose` (V2 plugin), fallback `docker-compose` (V1). Store the resolved command in config so subsequent calls are consistent.
- Ensure units are system-level (depend on `docker.service`, not rootless Podman semantics).

### 3) Preflight validation before deployment
- In `RunDeployment`:
  - Assert `systemctl is-active docker.service` succeeds; fail with a directive to start/enable Docker otherwise.
  - Verify compose availability via `docker compose version` or `docker-compose --version`; fail if neither works.
  - Run `docker compose config --quiet` (or V1 equivalent) to validate compose files before unit creation.
  - If NFS is configured, run `findmnt {mountPoint}` and abort with guidance if missing.

### 4) Migration from legacy mount units
- Add `migrateSystemdMountToFstab(cfg, ui, mountPoint)` in `internal/steps/nfs.go` (or a dedicated helper):
  - Detect `{escaped}.mount` (and related automount) under `/etc/systemd/system/`.
  - Stop and disable the unit, extract source/options, write fstab via `createFstabEntry`, remove the unit file, reload systemd daemons.
  - Re-run `mount -a` and `findmnt` to verify the migration.

## Testing Expectations
- Add unit tests for escaping helper and fstab duplicate detection.
- For integration paths, add targeted tests where feasible; otherwise, document manual test commands.
- Before merging: run `make test` (minimum) from `homelab-setup/`. Lint/format if touched Go files.

## Deliverables
- Updated Go implementation per above with passing tests.
- Clear error messages and inline comments where behavior changed.
- Brief doc note if runtime behavior differs from Podman-era flow.

Execute precisely. I will review every diff.
