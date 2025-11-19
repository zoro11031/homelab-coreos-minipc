# CLAUDE.md - AI Assistant Guide

**Last Updated**: 2025-11-17
**Purpose**: Comprehensive guide for AI assistants working with this codebase

---

## Project Overview

This repository contains BlueBuild configuration for building custom Fedora CoreOS images for NAB9 mini PCs. It consists of:

1. **Custom CoreOS Image** - BlueBuild-based image layering on UBlue uCore
2. **Setup CLI Tool Binary** - Compiled `homelab-setup` binary bundled in the image (source maintained separately at [plex-migration-homelab/homelab-setup](https://github.com/plex-migration-homelab/homelab-setup))
3. **Infrastructure as Code** - Butane/Ignition configs, systemd units, compose stacks

**Key Services**: Plex, Jellyfin (hardware transcoding), Nextcloud, Immich, Overseerr, Nginx Proxy Manager
**Architecture**: Intel QuickSync GPU, WireGuard VPN tunnel to VPS, NFS-backed media storage

**Note**: The Go source code for the `homelab-setup` CLI has been moved to a dedicated repository. This repo only contains the compiled binary and image configuration.

---

## Repository Structure

```
homelab-coreos-minipc/
├── recipes/                    # BlueBuild image recipes
│   ├── recipe.yml              # Main recipe (base: ucore, modules)
│   ├── packages.yml            # Package installation manifest
│   └── systemd.yml             # Systemd unit definitions
│
├── files/                      # Files bundled into image
│   ├── scripts/                # Build-time scripts (RPM Fusion)
│   ├── system/                 # Filesystem overlay (/etc, /usr)
│   │   ├── etc/systemd/system/ # Compose & WireGuard services
│   │   ├── etc/profile.d/      # Intel VAAPI environment
│   │   └── usr/bin/            # Compiled homelab-setup binary (auto-updated via GitHub Actions)
│   └── setup_scripts/          # Legacy bash setup scripts
│
├── ignition/                   # CoreOS first-boot provisioning
│   ├── config.bu.template      # Butane config template
│   └── transpile.sh            # Butane→Ignition transpiler
│
├── docs/                       # Documentation
│   ├── getting-started.md      # Quick setup guide
│   ├── reference/              # CLI manual, Ignition docs
│   ├── testing/                # QA checklists
│   └── dev/                    # Devcontainer, CI pipeline docs
│
├── .github/workflows/          # CI/CD
│   ├── build.yml               # BlueBuild image builds (daily @ 06:00 UTC)
│   └── build-homelab-setup.yml # Fetches & builds binary from plex-migration-homelab/homelab-setup
│
├── .vscode/                    # VS Code configuration
└── modules/                    # Custom BlueBuild modules
```

---

## Technology Stack

### Primary Technologies

**This Repository**: BlueBuild YAML configuration, Butane/Ignition configs, shell scripts

**homelab-setup CLI** (separate repo: [plex-migration-homelab/homelab-setup](https://github.com/plex-migration-homelab/homelab-setup)):
- Go 1.23.3
- Dependencies: cobra, fatih/color, golang.org/x/term

### Infrastructure

- **Base OS**: Fedora CoreOS (UBlue uCore variant)
- **Image Build**: BlueBuild v1.8 (YAML-based recipes)
- **Provisioning**: Butane/Ignition (FCOS 1.4.0)
- **Container Runtime**: Podman (primary) or Docker
- **VPN**: WireGuard
- **Storage**: NFS client
- **GPU**: Intel VAAPI (media-driver, libva, ffmpeg)

### CI/CD

- **GitHub Actions**: Image builds, binary compilation
- **Cosign**: Image signing (optional, via secrets)
- **GHCR**: GitHub Container Registry (`ghcr.io/zoro11031/homelab-coreos-minipc`)

---

## Development Workflows

### Working with This Repository

This repo contains BlueBuild configuration only. For Go development on the `homelab-setup` CLI, see the separate repository at [plex-migration-homelab/homelab-setup](https://github.com/plex-migration-homelab/homelab-setup).

**Typical workflow here**:
1. Edit BlueBuild recipes (`recipes/*.yml`)
2. Modify system overlays (`files/system/`)
3. Update Butane configs (`ignition/*.bu.template`)
4. GitHub Actions automatically fetches and builds the latest homelab-setup binary

### Common Development Tasks

#### Building the Custom Image

**Local Testing** (requires BlueBuild CLI):
```bash
# Install BlueBuild CLI first
# See: https://blue-build.org/learn/getting-started/

# Build recipe
bluebuild build recipes/recipe.yml

# Test in VM (virt-manager recommended)
# See: docs/testing/virt-manager-qa.md
```

**CI/CD** (automatic):
- Push to main → triggers build
- Daily @ 06:00 UTC → rebuilds image
- PR → test build (no push)

### Code Review Checklist

**Before Committing** (this repo):
1. Validate YAML syntax in `recipes/*.yml`
2. Test Butane config transpilation: `cd ignition && ./transpile.sh`
3. Verify file paths in `files/` match systemd unit expectations
4. Check GitHub Actions workflow syntax

**For Go CLI Development** (separate repo):
- See [plex-migration-homelab/homelab-setup](https://github.com/plex-migration-homelab/homelab-setup) for development guidelines

**File Operations**:
- Use `Read` tool for reading files
- Use `Edit` tool for modifying files
- Use `Write` tool only for NEW files (prefer editing existing)
- Never create unnecessary documentation files

---

## Code Conventions

### Naming Conventions

**Files**:
- Go: `snake_case.go`, `*_test.go`
- Shell: `kebab-case.sh`
- YAML: `kebab-case.yml`

**Go Identifiers**:
- Exported: `PascalCase`
- Unexported: `camelCase`
- Constants: `PascalCase` or `SCREAMING_SNAKE_CASE`

**Configuration Keys**:
- All uppercase with underscores: `HOMELAB_USER`, `NFS_SERVER`, `CONTAINER_RUNTIME`

**Shell Variables**:
- Exported/config: `UPPERCASE=value`
- Internal: `lowercase=value`

### Code Style

**EditorConfig** (`.editorconfig`):
```ini
[*.go]
indent_style = tab
indent_size = 4

[*.{yml,yaml,md}]
indent_style = space
indent_size = 2

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
```

**VS Code Settings** (`.vscode/settings.json`):
- Linter: `golangci-lint`
- Format on save: enabled
- Organize imports on save: enabled
- Build flags: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`
- Rulers: 100, 120 columns

### Error Handling Patterns

**Standard Pattern**:
```go
if err != nil {
    return fmt.Errorf("descriptive context: %w", err)
}
```

**System Operations**:
```go
cmd := exec.Command("sudo", "-n", "mkdir", "-p", path)
if output, err := cmd.CombinedOutput(); err != nil {
    return fmt.Errorf("failed to create directory %s: %w\nOutput: %s",
        path, err, string(output))
}
```

**User-Facing Errors**:
```go
// Use UI methods for colored output
ui.Error("Operation failed: " + err.Error())
ui.Warning("This is a warning")
ui.Success("Operation completed")
ui.Info("Information message")
```

### Validation

**Always validate user input** using `internal/common` validators:

```go
import "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"

// IP addresses (IPv4 only)
if err := common.ValidateIP("192.168.1.1"); err != nil {
    return err
}

// Ports (1-65535)
if err := common.ValidatePort("8080"); err != nil {
    return err
}

// CIDR blocks
if err := common.ValidateCIDR("10.0.0.0/24"); err != nil {
    return err
}

// Paths (absolute only)
if err := common.ValidatePath("/srv/containers"); err != nil {
    return err
}

// Safe paths (no shell metacharacters) - USE THIS for system commands
if err := common.ValidateSafePath(userInput); err != nil {
    return err
}
```

**Available Validators** (`homelab-setup/internal/common/validation.go`):
- `ValidateIP(ip string)` - IPv4 addresses
- `ValidatePort(port string)` - 1-65535
- `ValidateCIDR(cidr string)` - IPv4 CIDR blocks
- `ValidatePath(path string)` - Absolute paths
- `ValidateSafePath(path string)` - Paths safe for system commands (no metacharacters)
- `ValidateUsername(username string)` - Alphanumeric + dash
- `ValidateDomain(domain string)` - FQDN validation
- `ValidateTimezone(tz string)` - Timezone strings

---

## Testing Practices

### Test Organization

**File Location**: Co-located with source (`*_test.go`)

**Test Structure** (table-driven):
```go
func TestValidateIP(t *testing.T) {
    tests := []struct {
        name    string
        ip      string
        wantErr bool
    }{
        {"valid IPv4", "192.168.1.1", false},
        {"invalid - empty", "", true},
        {"invalid - too high", "256.1.1.1", true},
        {"invalid - IPv6", "::1", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateIP(tt.ip)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateIP() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Coverage

**Current Coverage** (as of 2025-11-17):
- `internal/common`: 100+ test cases (IP, port, CIDR, path, username, domain, timezone)
- `internal/config`: Config file operations, markers
- `internal/steps`: Individual setup steps
- `internal/system`: System operations

**Running Coverage**:
```bash
make test-coverage
# Opens: coverage.html
```

### Security Testing

**Always test for**:
1. Command injection via paths (use `ValidateSafePath`)
2. Shell metacharacter filtering
3. Invalid IP addresses/CIDR blocks
4. Port ranges (1-65535)
5. Empty/nil inputs
6. Overly long inputs

---

## Build & Deployment

### Binary Build (Automated)

**CI/CD Binary Build** (`.github/workflows/build-homelab-setup.yml`):
1. Triggered: Daily at 02:00 UTC, manual dispatch, or workflow file changes
2. Checks out: `plex-migration-homelab/homelab-setup` to temp dir
3. Runs tests: `go test ./... -v`
4. Builds binary: `make build`
5. Copies to: `files/system/usr/bin/homelab-setup`
6. Auto-commits if changed (using `GH_PAT` secret)
7. Uploads artifact (30-day retention)

**Manual Trigger**:
- Go to Actions → "Build homelab-setup Binary" → "Run workflow"

### Image Build

**BlueBuild Recipe** (`recipes/recipe.yml`):
```yaml
base-image: ghcr.io/ublue-os/ucore
image-version: stable

modules:
  - type: script
    scripts: [install-rpmfusion-release.sh]
  - from-file: packages.yml       # Package installation
  - type: files                   # Filesystem overlay
    files:
      - source: system
        destination: /
  - from-file: systemd.yml        # Service units
  - type: signing                 # Cosign signing
```

**Build Schedule**:
- **Daily**: 06:00 UTC (20 min after UBlue upstream builds)
- **On Push**: Automatic (ignores `*.md` changes)
- **On PR**: Test build (no push)
- **Manual**: `workflow_dispatch`

**Build Output**:
- Registry: `ghcr.io/zoro11031/homelab-coreos-minipc:latest`
- Tags: `latest`, git commit SHA
- Signing: Cosign (if `SIGNING_SECRET` set)

### First-Boot Deployment

**Process**:
1. **Install FCOS** with Ignition from `ignition/config.bu.template`
2. **First Boot**: Auto-rebase to custom image via systemd units
3. **Post-Rebase**: SSH as `core`, run `~/setup/homelab-setup`
4. **Interactive Setup**: Wizard configures user, WireGuard, NFS, containers

**Ignition Template** (`ignition/config.bu.template`):
- Creates `core` user with SSH keys
- Sets password hash
- Enables auto-rebase service
- Creates directories (`~/setup/`)

---

## Configuration Management

### Config File Format

**Location**: `~/.homelab-setup.conf`
**Format**: INI-style (simple `key=value`)

**Example**:
```ini
CONTAINER_RUNTIME=podman
HOMELAB_USER=containeruser
PUID=1001
PGID=1001
TZ=America/Chicago
NFS_SERVER=192.168.7.10
NFS_MEDIA_PATH=/volume1/media
WG_SERVER_ENDPOINT=vpn.example.com:51820
```

### Completion Markers

**Location**: `~/.local/homelab-setup/`
**Files**:
- `preflight-complete`
- `user-setup-complete`
- `directory-setup-complete`
- `wireguard-setup-complete`
- `nfs-setup-complete`
- `container-setup-complete`
- `service-deployment-complete`

**Usage**: Touch files to mark steps complete, remove to re-run

### Preseed Support

**Environment Variables** (auto-seeds config):
- `HOMELAB_USER` - Service account username
- `SETUP_USER` - Legacy key (maps to `HOMELAB_USER`)

**Example**:
```bash
HOMELAB_USER=containeruser ./homelab-setup
# Skips user input prompt for username
```

---

## Key Files Reference

### Must-Read Files for New Contributors

| File | Purpose | Lines |
|------|---------|-------|
| `recipes/recipe.yml` | Image build recipe | ~25 |
| `recipes/packages.yml` | Package installation manifest | ~50 |
| `recipes/systemd.yml` | Systemd unit definitions | ~30 |
| `files/system/` | Filesystem overlay structure | - |
| `ignition/config.bu.template` | First-boot configuration | ~100 |
| `.github/workflows/build.yml` | Image CI/CD | ~45 |
| `.github/workflows/build-homelab-setup.yml` | Binary build from upstream | ~70 |
| `docs/getting-started.md` | User quickstart | ~200 |

### Build Configuration

| File | Purpose |
|------|---------|
| `recipes/*.yml` | BlueBuild image recipes |
| `ignition/*.bu.template` | Butane configuration templates |
| `.editorconfig` | Cross-editor formatting |
| `.vscode/settings.json` | VS Code configuration |
| `.github/workflows/build.yml` | Image build CI/CD |
| `.github/workflows/build-homelab-setup.yml` | Binary build from upstream |

---

## Important Patterns to Follow

### 1. BlueBuild Recipe Structure

```yaml
# recipes/recipe.yml
base-image: ghcr.io/ublue-os/ucore
image-version: stable

modules:
  - type: script
    scripts:
      - install-rpmfusion-release.sh
  - from-file: packages.yml
  - type: files
    files:
      - source: system
        destination: /
  - from-file: systemd.yml
```

### 2. Systemd Unit Pattern

```ini
# files/system/etc/systemd/system/example.service
[Unit]
Description=Example Service
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/srv/containers/example
ExecStart=/usr/bin/podman-compose up -d
ExecStop=/usr/bin/podman-compose down

[Install]
WantedBy=multi-user.target
```

### 3. Butane Configuration Pattern

```yaml
# ignition/config.bu.template
variant: fcos
version: 1.4.0
passwd:
  users:
    - name: core
      ssh_authorized_keys:
        - "ssh-ed25519 AAAA..."
      password_hash: "$6$..."
storage:
  directories:
    - path: /home/core/setup
      mode: 0755
systemd:
  units:
    - name: auto-rebase.service
      enabled: true
```

---

## Common Pitfalls to Avoid

### BlueBuild Configuration

1. **YAML syntax validation**
   - Use 2-space indentation for YAML files
   - Validate with BlueBuild CLI before committing
   - Test recipe changes locally first

2. **File paths in recipes**
   - Use absolute paths from image root (`/etc/`, `/usr/`)
   - Ensure source files exist in `files/` directory
   - Match systemd unit paths with actual file locations

3. **Module ordering matters**
   - Scripts run before file copying
   - Packages install before systemd units
   - Check module dependencies

### Ignition/Butane

1. **Butane transpilation**
   - Always test with `./ignition/transpile.sh`
   - Validate FCOS version compatibility (1.4.0)
   - Check password hash format (`$6$...`)

2. **First-boot timing**
   - Units may run before network is ready
   - Use `After=network-online.target`
   - Test rebase units in VM

### Documentation

1. **Keep docs synchronized**
   - Update README.md when changing architecture
   - Document new systemd units in reference docs
   - Update AGENTS.md and CLAUDE.md for major changes

---

## Quick Reference Commands

### BlueBuild Development

```bash
# Test Butane transpilation
cd ignition/
./transpile.sh

# Build image locally (requires BlueBuild CLI)
bluebuild build recipes/recipe.yml

# Validate YAML syntax
yamllint recipes/*.yml
```

### Binary Management

```bash
# Trigger binary rebuild (GitHub Actions)
# Go to: Actions → "Build homelab-setup Binary" → "Run workflow"

# Check current binary
ls -lh files/system/usr/bin/homelab-setup
```

### Git Workflow

```bash
# Create feature branch (must start with claude/)
git checkout -b claude/feature-name-<session-id>

# Commit changes
git add .
git commit -m "feat: Add feature description"

# Push (with retry on network errors)
git push -u origin claude/feature-name-<session-id>
```

### Testing Image Locally

```bash
# Build image (requires BlueBuild CLI)
bluebuild build recipes/recipe.yml

# Test in virt-manager
# See: docs/testing/virt-manager-qa.md
```

### Debugging

```bash
# Check binary version (on deployed system)
/usr/bin/homelab-setup version

# Check setup status
/usr/bin/homelab-setup status

# View systemd units
systemctl list-units | grep -E "podman|docker|wg-quick"

# Check image version
rpm-ostree status

# View rebase logs
journalctl -u auto-rebase.service
```

---

## Getting Help

### Documentation

- **Getting Started**: `docs/getting-started.md`
- **CLI Manual**: `docs/reference/homelab-setup-cli.md`
- **Ignition Guide**: `docs/reference/ignition.md`
- **Testing**: `docs/testing/virt-manager-qa.md`
- **Go Implementation**: `homelab-setup/README.md`

### External Resources

- **BlueBuild Docs**: https://blue-build.org/
- **Fedora CoreOS**: https://docs.fedoraproject.org/en-US/fedora-coreos/
- **Butane Configs**: https://coreos.github.io/butane/
- **UBlue**: https://universal-blue.org/

### Project Structure

- **Main Branch**: Production-ready code
- **Feature Branches**: `claude/<description>-<session-id>`
- **PR Process**: Test build runs automatically, requires passing tests

---

## Change Log

| Date | Change |
|------|--------|
| 2025-11-19 | Migrated homelab-setup Go source to separate repo (plex-migration-homelab/homelab-setup); this repo now contains only BlueBuild config and compiled binary |
| 2025-11-17 | Initial CLAUDE.md creation |

---

**For AI Assistants**: This repository is now focused on BlueBuild image configuration only. The Go CLI source code has been moved to [plex-migration-homelab/homelab-setup](https://github.com/plex-migration-homelab/homelab-setup). When working with this repo, focus on YAML recipes, Butane configs, systemd units, and shell scripts. For Go development, refer to the separate repository.
