# Handoff: Phase 3 Implementation - UBlue uCore Homelab Setup

## 🎯 Mission

Implement Phase 3 of the Go rewrite: **Setup Steps Module**. This includes the core setup logic that was previously in bash scripts, now implemented as clean, testable Go packages.

## 📍 Current State

### Git Branch
**Branch**: `claude/rewrite-setup-scripts-go-011CV2fGyT3BkCy9mZxCC3At`
**Latest Commit**: `8aa43d0` - Add comprehensive final review prompt for Codex
**Status**: Clean working directory, all tests passing (49/49)

### What's Complete ✅

**Phase 1 - Foundation** (100% Complete)
- `internal/config/config.go` - Key-value config management with atomic writes
- `internal/config/markers.go` - Completion tracking with path traversal protection
- `internal/ui/output.go` - Colored terminal output
- `internal/ui/prompts.go` - Interactive prompts (survey library)
- `internal/common/validation.go` - Input validators (IP, port, path, username, domain, timezone)
- `pkg/version/version.go` - Version information with build-time injection
- `cmd/homelab-setup/main.go` - Cobra CLI framework setup

**Phase 2 - System Operations** (100% Complete)
- `internal/system/packages.go` - rpm-ostree package detection and management
- `internal/system/services.go` - Systemd service operations (enable, start, status, logs)
- `internal/system/users.go` - User/group management, subuid/subgid for rootless containers
- `internal/system/filesystem.go` - File operations with sudo (atomic writes, safety checks)
- `internal/system/network.go` - Connectivity tests, port scanning, NFS validation
- `internal/system/containers.go` - Container runtime detection (podman/docker), rootless checks

**Infrastructure**
- `.devcontainer/` - VS Code dev container with Go 1.24
- `.vscode/` - Tasks, launch configs, settings for development
- `Makefile` - Build system with test, lint, coverage targets
- `docs/go-rewrite-plan.md` - Original architecture plan
- `docs/future-refactoring.md` - Deferred improvements (non-critical)
- `docs/final-review-prompt.md` - Codex review template

### Security Hardening Applied 🔒

Recent fixes ensure production-readiness:
1. **Atomic writes** in config.go (temp file → sync → rename)
2. **Path traversal prevention** in markers.go (validates marker names)
3. **Critical path protection** in filesystem.go (blocks rm -rf on /, /etc, /usr, etc.)
4. **Type assertion safety** (panic prevention in GetOwner, IsMount, BackupFile)
5. **Ownership security** in WriteFile (chown to root after sudo mv)
6. **Non-interactive sudo** (-n flag) in all systemd operations
7. **Rootless container detection** fixed for Docker (checks SecurityOptions)

### Test Coverage
- **49 tests passing** across 3 packages
- Unit tests for all validation, config, and system operation modules
- No tests for UI/main (acceptable - interactive/integration)

## 🚀 Phase 3 - Your Mission

### Overview

Implement the setup steps that orchestrate system operations into complete setup flows. These were previously bash functions in the original scripts.

### Files to Create

Create these new files in `internal/steps/`:

#### 1. `internal/steps/preflight.go` (~200 lines)
**Purpose**: System validation checks before setup begins

**Key Functions**:
```go
type PreflightChecker struct {
    packages *system.PackageManager
    network  *system.Network
    ui       *ui.UI
}

func (p *PreflightChecker) CheckRpmOstree() error
func (p *PreflightChecker) CheckRequiredPackages() error
func (p *PreflightChecker) CheckNetworkConnectivity() error
func (p *PreflightChecker) CheckNFSServer(host string) error
func (p *PreflightChecker) RunAll() error
```

**Requirements**:
- Check system is rpm-ostree based
- Verify required packages installed (nfs-utils, etc.)
- Test network connectivity
- Validate NFS server accessibility (if configured)
- Provide clear, actionable error messages
- Use UI for progress output

**Reference**: Original bash `check_prerequisites()`, `check_nfs_connectivity()`

---

#### 2. `internal/steps/user.go` (~250 lines)
**Purpose**: User and group configuration for homelab services

**Key Functions**:
```go
type UserConfigurator struct {
    users   *system.UserManager
    config  *config.Config
    ui      *ui.UI
    markers *config.Markers
}

func (u *UserConfigurator) PromptForUser() (string, error)
func (u *UserConfigurator) ValidateUser(username string) error
func (u *UserConfigurator) CreateUserIfNeeded(username string) error
func (u *UserConfigurator) ConfigureSubuidSubgid(username string) error
func (u *UserConfigurator) SetupShell(username string, shell string) error
func (u *UserConfigurator) Run() error
```

**Requirements**:
- Prompt for homelab username (or use existing user)
- Validate username exists or create with proper groups
- Configure subuid/subgid mappings for rootless containers (100000:65536)
- Set up shell and home directory
- Save to config: `HOMELAB_USER`
- Create marker: `user-configured`
- Skip if marker exists (idempotent)

**Reference**: Original bash `setup_homelab_user()`, `configure_subuid_subgid()`

---

#### 3. `internal/steps/directory.go` (~200 lines)
**Purpose**: Create directory structure for homelab services

**Key Functions**:
```go
type DirectorySetup struct {
    fs      *system.FileSystem
    config  *config.Config
    ui      *ui.UI
    markers *config.Markers
}

func (d *DirectorySetup) PromptForBaseDir() (string, error)
func (d *DirectorySetup) CreateBaseStructure(baseDir, owner string) error
func (d *DirectorySetup) CreateServiceDirs(baseDir, owner string, services []string) error
func (d *DirectorySetup) Run() error
```

**Requirements**:
- Prompt for base directory (default: `/mnt/homelab`)
- Create structure:
  ```
  /mnt/homelab/
    ├── config/       (owner:owner 0755)
    ├── data/         (owner:owner 0755)
    ├── compose/      (owner:owner 0755)
    └── services/     (owner:owner 0755)
        ├── adguard/
        ├── caddy/
        ├── etc...
  ```
- Set ownership and permissions correctly
- Save to config: `HOMELAB_BASE_DIR`
- Create marker: `directories-created`
- Skip if marker exists

**Reference**: Original bash `setup_directories()`

---

#### 4. `internal/steps/nfs.go` (~300 lines)
**Purpose**: NFS mount configuration (optional)

**Key Functions**:
```go
type NFSConfigurator struct {
    fs      *system.FileSystem
    network *system.Network
    config  *config.Config
    ui      *ui.UI
    markers *config.Markers
}

func (n *NFSConfigurator) PromptForNFS() (bool, error)
func (n *NFSConfigurator) PromptForNFSDetails() (host, export string, err error)
func (n *NFSConfigurator) ValidateNFSConnection(host, export string) error
func (n *NFSConfigurator) CreateNFSMount(host, export, mountPoint string) error
func (n *NFSConfigurator) AddToFstab(host, export, mountPoint string) error
func (n *NFSConfigurator) Run() error
```

**Requirements**:
- Prompt: "Use NFS mount?" (optional)
- If yes: prompt for NFS server IP/hostname and export path
- Validate NFS server is reachable (showmount -e)
- Create mount point with sudo
- Add to /etc/fstab with proper options (nfsvers=4.2, etc.)
- Mount the share
- Save to config: `NFS_SERVER`, `NFS_EXPORT`, `NFS_MOUNT_POINT`
- Create marker: `nfs-configured` or `nfs-skipped`
- Handle "skip NFS" gracefully

**Reference**: Original bash `setup_nfs_mount()`, `add_to_fstab()`

---

#### 5. `internal/steps/containers.go` (~250 lines)
**Purpose**: Container runtime setup and validation

**Key Functions**:
```go
type ContainerSetup struct {
    containers *system.ContainerManager
    config     *config.Config
    ui         *ui.UI
    markers    *config.Markers
}

func (c *ContainerSetup) DetectRuntime() (system.ContainerRuntime, error)
func (c *ContainerSetup) ValidateRootless(runtime system.ContainerRuntime, user string) error
func (c *ContainerSetup) ConfigureRuntime(runtime system.ContainerRuntime) error
func (c *ContainerSetup) Run() error
```

**Requirements**:
- Auto-detect container runtime (podman preferred, docker fallback)
- Validate rootless mode is enabled for the homelab user
- Provide instructions if rootless not configured
- Test basic container operations (pull busybox, run, rm)
- Save to config: `CONTAINER_RUNTIME` (podman/docker)
- Create marker: `containers-configured`

**Reference**: Original bash `setup_container_runtime()`, `check_rootless_containers()`

---

#### 6. `internal/steps/deployment.go` (~200 lines)
**Purpose**: Deploy compose files and start services

**Key Functions**:
```go
type Deployment struct {
    containers *system.ContainerManager
    fs         *system.FileSystem
    services   *system.ServiceManager
    config     *config.Config
    ui         *ui.UI
    markers    *config.Markers
}

func (d *Deployment) CopyComposeFiles(sourceDir, destDir string) error
func (d *Deployment) GenerateEnvFile(service string) error
func (d *Deployment) StartServices(services []string) error
func (d *Deployment) Run() error
```

**Requirements**:
- Copy compose files from repo to homelab directory
- Generate .env files for services (if needed)
- Start services using podman-compose or docker-compose
- Validate services are running
- Create marker: `services-deployed`

**Reference**: Original bash `deploy_services()`, `start_compose_stack()`

---

#### 7. `internal/steps/wireguard.go` (~300 lines)
**Purpose**: WireGuard VPN setup (optional)

**Key Functions**:
```go
type WireGuardSetup struct {
    packages *system.PackageManager
    services *system.ServiceManager
    fs       *system.FileSystem
    network  *system.Network
    config   *config.Config
    ui       *ui.UI
    markers  *config.Markers
}

func (w *WireGuardSetup) PromptForWireGuard() (bool, error)
func (w *WireGuardSetup) InstallWireGuard() error
func (w *WireGuardSetup) GenerateKeys() error
func (w *WireGuardSetup) PromptForConfig() (*WGConfig, error)
func (w *WireGuardSetup) WriteConfig(cfg *WGConfig) error
func (w *WireGuardSetup) EnableService() error
func (w *WireGuardSetup) Run() error
```

**Requirements**:
- Prompt: "Set up WireGuard VPN?" (optional)
- If yes: install wireguard-tools package
- Generate private/public keys securely
- Prompt for: interface IP, listen port, peer public keys
- Write /etc/wireguard/wg0.conf
- Enable and start wg-quick@wg0 service
- Save to config: `WIREGUARD_ENABLED`, `WIREGUARD_INTERFACE`
- Create marker: `wireguard-configured` or `wireguard-skipped`

**Reference**: Original bash `setup_wireguard()`, `generate_wireguard_keys()`

---

### Architecture Guidelines

#### Package Structure
```
internal/steps/
├── preflight.go      # System validation
├── user.go           # User configuration
├── directory.go      # Directory structure
├── nfs.go           # NFS mounts (optional)
├── containers.go     # Container runtime
├── deployment.go     # Service deployment
├── wireguard.go      # VPN setup (optional)
└── steps_test.go     # Integration tests
```

#### Common Patterns

**1. Manager Pattern**
```go
type StepName struct {
    // Dependencies (system managers)
    dependency1 *system.SomeManager
    dependency2 *system.AnotherManager

    // Core dependencies (always needed)
    config  *config.Config
    ui      *ui.UI
    markers *config.Markers
}

func NewStepName(deps ...) *StepName {
    return &StepName{
        dependency1: deps.D1,
        config:      deps.Config,
        ui:          deps.UI,
        markers:     deps.Markers,
    }
}
```

**2. Idempotent Execution**
```go
func (s *StepName) Run() error {
    // Check if already complete
    exists, err := s.markers.Exists("step-name-complete")
    if err != nil {
        return fmt.Errorf("failed to check marker: %w", err)
    }
    if exists {
        s.ui.Info("Step already completed, skipping")
        return nil
    }

    // Do the work...
    if err := s.doWork(); err != nil {
        return fmt.Errorf("failed to do work: %w", err)
    }

    // Mark complete
    if err := s.markers.Create("step-name-complete"); err != nil {
        return fmt.Errorf("failed to create marker: %w", err)
    }

    s.ui.Success("Step completed successfully")
    return nil
}
```

**3. Interactive Prompts**
```go
// Use existing UI methods
username, err := s.ui.PromptInput("Enter homelab username", "homelab")
if err != nil {
    return fmt.Errorf("failed to prompt: %w", err)
}

// Validate immediately
if err := common.ValidateUsername(username); err != nil {
    return fmt.Errorf("invalid username: %w", err)
}
```

**4. Error Handling**
```go
// Always wrap errors with context
if err := s.fs.EnsureDirectory(path, owner, 0755); err != nil {
    return fmt.Errorf("failed to create directory %s: %w", path, err)
}

// Provide actionable messages to users
if err := validateNFS(host); err != nil {
    s.ui.Error("NFS server unreachable")
    s.ui.Info("Please check:")
    s.ui.Info("  1. Server is powered on")
    s.ui.Info("  2. Network connectivity")
    s.ui.Info("  3. NFS service is running")
    return err
}
```

**5. Configuration Management**
```go
// Save important values to config
if err := s.config.Set("HOMELAB_USER", username); err != nil {
    return fmt.Errorf("failed to save config: %w", err)
}

// Load from config
username := s.config.GetOrDefault("HOMELAB_USER", "homelab")
```

### Testing Requirements

Create `internal/steps/steps_test.go` with:

1. **Unit Tests** for each step's public methods
2. **Mock Dependencies** using interfaces where possible
3. **Idempotency Tests** - verify marking/skipping works
4. **Error Cases** - test validation failures, permission errors
5. **Integration Tests** - test step orchestration

Example:
```go
func TestUserConfiguratorRun(t *testing.T) {
    // Setup mock dependencies
    mockUsers := &MockUserManager{}
    mockConfig := config.New("/tmp/test.conf")
    mockMarkers := config.NewMarkers("/tmp/markers")
    mockUI := ui.New(os.Stdout)

    uc := NewUserConfigurator(mockUsers, mockConfig, mockMarkers, mockUI)

    // Test first run
    err := uc.Run()
    assert.NoError(t, err)

    // Verify marker created
    exists, _ := mockMarkers.Exists("user-configured")
    assert.True(t, exists)

    // Test second run (should skip)
    err = uc.Run()
    assert.NoError(t, err)
    // Should not error, should skip
}
```

### Important Notes

#### Security Considerations
- All sudo operations already use `-n` flag (fail fast without password)
- File operations use atomic writes where appropriate
- Critical paths are protected in RemoveDirectory
- Validate ALL user input before system operations

#### Idempotency is Critical
- Every step must be safely re-runnable
- Use markers to track completion
- Check markers at start of Run()
- Provide clear "already completed" messages

#### User Experience
- Use colored output (Info, Success, Warning, Error)
- Show progress: "Creating directories... " → "✓ Created"
- Provide helpful error messages with next steps
- Ask before destructive operations

#### Configuration Keys (Standardize)
Use these keys in config.Config:
- `HOMELAB_USER` - Main user for services
- `HOMELAB_BASE_DIR` - Base directory path
- `NFS_SERVER` - NFS server hostname/IP
- `NFS_EXPORT` - NFS export path
- `NFS_MOUNT_POINT` - Local mount point
- `CONTAINER_RUNTIME` - "podman" or "docker"
- `WIREGUARD_ENABLED` - "true" or "false"
- `WIREGUARD_INTERFACE` - Interface name (e.g., "wg0")

#### Marker Names (Standardize)
Use these marker names:
- `user-configured`
- `directories-created`
- `nfs-configured` or `nfs-skipped`
- `containers-configured`
- `services-deployed`
- `wireguard-configured` or `wireguard-skipped`

## 📚 Reference Materials

### Key Files to Study

**System Operations (Your building blocks)**:
- `internal/system/filesystem.go` - File operations you'll use extensively
- `internal/system/users.go` - User management operations
- `internal/system/services.go` - Systemd service control
- `internal/system/network.go` - Network validation
- `internal/system/containers.go` - Container runtime operations

**Configuration & State**:
- `internal/config/config.go` - How to save/load config
- `internal/config/markers.go` - How to track step completion

**UI Interaction**:
- `internal/ui/prompts.go` - All available prompt methods
- `internal/ui/output.go` - Colored output methods

**Original Bash Scripts** (for reference):
- `scripts/setup.sh` - Main setup flow
- `scripts/setup-lib.sh` - Helper functions

### Go Architecture Plan
See `docs/go-rewrite-plan.md` for:
- Complete function mapping (bash → Go)
- Phase breakdown and timeline
- Architecture decisions

### Build & Test Commands

```bash
# Build
cd homelab-setup
make build

# Run tests
make test              # Non-verbose
make test-verbose      # Verbose with coverage
make test-coverage     # Generate HTML report

# Format & lint
make fmt
make vet
make lint              # Requires golangci-lint

# Run binary
./bin/homelab-setup version
```

### Development Environment

**VS Code Dev Container** (Recommended):
1. Open workspace in VS Code
2. Click "Reopen in Container"
3. Full Go tooling automatically installed

**Manual Setup**:
- Go 1.23 or higher
- golangci-lint (optional, for linting)
- Passwordless sudo configured (for testing)

## 🎯 Success Criteria

Phase 3 is complete when:

1. ✅ All 7 step modules implemented and tested
2. ✅ Tests passing (aim for >40 new tests)
3. ✅ Each step is idempotent (re-runnable safely)
4. ✅ Clear error messages with actionable guidance
5. ✅ Configuration saved to config file
6. ✅ Markers track completion state
7. ✅ Code follows existing patterns from Phase 1 & 2
8. ✅ No security vulnerabilities introduced

## 🚨 Common Pitfalls to Avoid

1. **Don't skip marker checks** - Always check and create markers
2. **Don't trust user input** - Validate everything with common.Validate*
3. **Don't ignore errors** - Wrap all errors with context
4. **Don't hardcode paths** - Use config for user-configurable values
5. **Don't forget sudo -n** - Already handled in system packages, but be aware
6. **Don't panic** - Use proper error returns, no type assertions without checks
7. **Don't block on prompts** - Provide defaults for automation
8. **Don't forget tests** - Test each method, especially error cases

## 🤝 Handoff Checklist

Before you start:
- [ ] Read this entire document
- [ ] Review `docs/go-rewrite-plan.md`
- [ ] Study existing system operations in `internal/system/`
- [ ] Check out the correct branch
- [ ] Run `make test` to verify baseline
- [ ] Review original bash scripts for business logic

During development:
- [ ] Create one step file at a time
- [ ] Write tests as you go
- [ ] Run tests frequently: `make test`
- [ ] Keep commits focused and well-described
- [ ] Ask questions if business logic is unclear

When complete:
- [ ] All tests passing: `make test-coverage`
- [ ] No golangci-lint warnings: `make lint`
- [ ] Code formatted: `make fmt`
- [ ] Commit all changes with clear message
- [ ] Push to branch
- [ ] Consider running Codex final review (use `docs/final-review-prompt.md`)

## 📞 Need Help?

**Documentation**:
- Architecture: `docs/go-rewrite-plan.md`
- Future work: `docs/future-refactoring.md`
- Review template: `docs/final-review-prompt.md`

**Code Examples**:
- Look at Phase 2 implementation in `internal/system/` for patterns
- Tests in `internal/system/system_test.go` show testing approach
- Config usage examples in `internal/config/config_test.go`

**Git History**:
```bash
# See recent changes
git log --oneline -10

# See specific commit
git show 8aa43d0
```

## 🎉 Ready to Code!

You have a solid foundation in Phase 1 & 2. Phase 3 is about orchestrating those building blocks into complete setup flows. The hardest work (system operations, security hardening) is done. Now you're building the user-facing setup experience.

**Start with `preflight.go`** - it's the simplest and sets the pattern for the rest.

Good luck! You've got this. 🚀

---

**Branch**: `claude/rewrite-setup-scripts-go-011CV2fGyT3BkCy9mZxCC3At`
**Starting Point**: All Phase 1 & 2 complete, 49 tests passing
**Your Mission**: Implement Phase 3 setup steps
**End Goal**: Complete, tested setup orchestration ready for Phase 4 (CLI)
