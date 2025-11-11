# Homelab Setup Scripts - Go Rewrite Plan

## Executive Summary

Rewrite the homelab setup bash scripts in Go to create a single compiled binary that is:
- **Dependency-free**: No Python, jq, or specific bash versions required
- **Type-safe**: Compile-time error checking
- **Testable**: Unit tests for all logic
- **Maintainable**: Clear package structure and error handling
- **Hybrid**: Can still shell out to existing bash scripts when needed

## Current State Analysis

### Existing Scripts Overview

| Script | Lines | Purpose | Complexity |
|--------|-------|---------|------------|
| `homelab-setup.sh` | ~520 | Main orchestrator with interactive menu | Medium |
| `common-functions.sh` | ~654 | Shared utilities and functions | High |
| `00-preflight-check.sh` | ~509 | System validation | Medium |
| `01-user-setup.sh` | ~477 | User/group configuration | Medium |
| `02-directory-setup.sh` | ~392 | Directory structure creation | Low |
| `03-wireguard-setup.sh` | ~400+ | VPN configuration | High |
| `04-nfs-setup.sh` | ~415 | NFS mount configuration | Medium |
| `05-container-setup.sh` | ~851 | Container stack selection & .env | High |
| `06-service-deployment.sh` | ~400+ | Service enablement and startup | Medium |
| `troubleshoot.sh` | ~400+ | Diagnostic tool | Medium |

**Total: ~4,600+ lines of bash**

### Key Functionality by Category

#### 1. **UI & User Interaction**
- Interactive menus (numbered choices, Y/N prompts)
- Colored output (info, success, warning, error)
- Progress bars and spinners
- Password prompts (hidden input)
- Input validation

#### 2. **Configuration Management**
- Read/write config file (`~/.homelab-setup.conf`)
- Load/save key-value pairs
- Marker files for tracking completion
- Environment variable generation

#### 3. **System Operations**
- Package detection (`rpm -q`)
- Service management (systemd)
- User/group management
- File system operations
- Network connectivity tests
- Mount point management

#### 4. **Container Operations**
- Runtime detection (podman/docker)
- Compose command detection
- Template file processing
- .env file generation
- Service discovery

#### 5. **Setup Steps**
Each step is independent but has dependencies:
```
00-preflight → 01-user → 02-directory → [03-wireguard] → 04-nfs → 05-container → 06-deployment
                                           (optional)
```

## Proposed Go Architecture

### Directory Structure

```
homelab-setup-go/
├── cmd/
│   └── homelab-setup/
│       └── main.go                      # CLI entry point
├── internal/
│   ├── config/
│   │   ├── config.go                    # Configuration management
│   │   ├── markers.go                   # Completion markers
│   │   └── state.go                     # State tracking
│   ├── ui/
│   │   ├── prompts.go                   # Interactive prompts
│   │   ├── output.go                    # Colored logging
│   │   ├── progress.go                  # Progress bars
│   │   └── menu.go                      # Menu rendering
│   ├── system/
│   │   ├── packages.go                  # Package detection
│   │   ├── services.go                  # Systemd service management
│   │   ├── users.go                     # User/group operations
│   │   ├── filesystem.go                # File operations
│   │   ├── network.go                   # Network tests
│   │   └── containers.go                # Container runtime detection
│   ├── steps/
│   │   ├── step.go                      # Step interface
│   │   ├── preflight.go                 # 00-preflight-check
│   │   ├── user.go                      # 01-user-setup
│   │   ├── directory.go                 # 02-directory-setup
│   │   ├── wireguard.go                 # 03-wireguard-setup
│   │   ├── nfs.go                       # 04-nfs-setup
│   │   ├── container.go                 # 05-container-setup
│   │   └── deployment.go                # 06-service-deployment
│   ├── troubleshoot/
│   │   └── troubleshoot.go              # Diagnostic tool
│   └── common/
│       ├── validation.go                # Input validators (IP, port, path)
│       ├── shell.go                     # Shell command execution
│       └── templates.go                 # Template rendering
├── pkg/
│   └── version/
│       └── version.go                   # Version info
├── scripts/
│   └── legacy/                          # Keep existing bash scripts for complex operations
│       ├── wireguard-keygen.sh
│       └── ... (other complex scripts)
├── go.mod
├── go.sum
├── Makefile                             # Build automation
└── README.md
```

### Core Interfaces

#### Step Interface
```go
type Step interface {
    Name() string
    Description() string
    PreCheck() error           // Check if prerequisites are met
    Run(ctx context.Context) error
    Verify() error            // Post-execution verification
    IsComplete() bool         // Check if already completed
}
```

#### Config Interface
```go
type Config interface {
    Get(key string) (string, error)
    Set(key string, value string) error
    GetOrDefault(key string, defaultValue string) string
    Exists(key string) bool
    Save() error
    Load() error
}
```

#### UI Interface
```go
type UI interface {
    Info(msg string)
    Success(msg string)
    Warning(msg string)
    Error(msg string)

    PromptYesNo(prompt string, defaultYes bool) (bool, error)
    PromptInput(prompt string, defaultValue string) (string, error)
    PromptPassword(prompt string) (string, error)
    PromptSelect(prompt string, options []string) (int, error)

    ShowProgress(current, total int, message string)
    StartSpinner(message string)
    StopSpinner()
}
```

## Function Mapping: Bash → Go

### common-functions.sh → Go Packages

| Bash Function | Go Package | Go Function | Notes |
|---------------|------------|-------------|-------|
| `log_info()` | `ui` | `UI.Info()` | |
| `log_success()` | `ui` | `UI.Success()` | |
| `log_warning()` | `ui` | `UI.Warning()` | |
| `log_error()` | `ui` | `UI.Error()` | |
| `prompt_yes_no()` | `ui` | `UI.PromptYesNo()` | |
| `prompt_input()` | `ui` | `UI.PromptInput()` | |
| `prompt_password()` | `ui` | `UI.PromptPassword()` | |
| `save_config()` | `config` | `Config.Set()` | |
| `load_config()` | `config` | `Config.Get()` | |
| `create_marker()` | `config` | `Markers.Create()` | |
| `check_marker()` | `config` | `Markers.Exists()` | |
| `validate_ip()` | `common` | `ValidateIP()` | |
| `validate_port()` | `common` | `ValidatePort()` | |
| `check_command()` | `system` | `CommandExists()` | |
| `check_package()` | `system` | `PackageInstalled()` | |
| `check_systemd_service()` | `system` | `ServiceExists()` | |
| `enable_service()` | `system` | `EnableService()` | |
| `start_service()` | `system` | `StartService()` | |
| `ensure_directory()` | `system` | `EnsureDirectory()` | |
| `test_connectivity()` | `system` | `TestConnectivity()` | |
| `detect_container_runtime()` | `system` | `DetectContainerRuntime()` | |
| `get_compose_command()` | `system` | `GetComposeCommand()` | |

### Setup Steps → Go Steps Package

Each bash script becomes a Go struct implementing the `Step` interface:

```go
// Example: 01-user-setup.sh → steps/user.go
type UserSetupStep struct {
    config config.Config
    ui     ui.UI
    system system.System
}

func (s *UserSetupStep) Name() string {
    return "user-setup"
}

func (s *UserSetupStep) Run(ctx context.Context) error {
    // Interactive user configuration
    // User creation/selection
    // Group management
    // Subuid/subgid setup
    // UID/GID detection
    return nil
}
```

## Go Libraries & Dependencies

### Required External Libraries

1. **CLI Framework**: `github.com/spf13/cobra`
   - Command-line interface and subcommands
   - Flag parsing
   - Help generation

2. **Interactive Prompts**: `github.com/AlecAivazis/survey/v2`
   - Yes/No prompts
   - Input with validation
   - Select menus
   - Password input (hidden)

3. **Configuration**: `github.com/spf13/viper`
   - Config file management
   - Environment variable support
   - Multiple format support

4. **Colored Output**: `github.com/fatih/color`
   - ANSI color output
   - Styled text
   - Cross-platform support

5. **Progress Bars**: `github.com/schollz/progressbar/v3`
   - Progress indicators
   - Customizable themes

6. **Systemd Integration**: `github.com/coreos/go-systemd/v22`
   - Native systemd operations
   - D-Bus communication

### Standard Library Usage

- `os/exec`: Running shell commands
- `os`: File system operations
- `os/user`: User/group operations
- `net`: Network validation and tests
- `text/template`: Template rendering
- `context`: Cancellation and timeouts
- `testing`: Unit tests

## Implementation Strategy: Hybrid Approach

### Phase 1: Core Infrastructure (Go)
✅ **Implement in Go first**
- Configuration management
- UI/prompts
- Logging and output
- Common validators
- System checks (package, service, command detection)

### Phase 2: Simple Steps (Go)
✅ **Implement in Go**
- Preflight checks (mostly system queries)
- User setup (user/group operations)
- Directory setup (mkdir, chown, chmod)
- NFS setup (systemd unit generation)
- Service deployment (systemd operations)

### Phase 3: Complex Operations (Shell Out)
⚠️ **Keep as bash scripts, call from Go**
- WireGuard key generation
  - Uses `wg genkey`, `wg pubkey`, complex crypto
  - Keep existing `generate-keys.sh`
  - Call from Go: `exec.Command("bash", "/path/to/generate-keys.sh")`

- Container image pulls
  - `podman-compose pull` can be slow/complex
  - Better to shell out to existing compose commands

- Complex systemd operations
  - `systemctl daemon-reload`, `systemctl enable/start`
  - Can use `go-systemd` library OR shell out

### Shell Command Execution Pattern

```go
// internal/common/shell.go
func RunScript(scriptPath string, args ...string) (string, error) {
    cmd := exec.Command("bash", scriptPath)
    cmd.Args = append(cmd.Args, args...)

    output, err := cmd.CombinedOutput()
    return string(output), err
}

// Usage in step
func (s *WireGuardStep) generateKeys() error {
    scriptPath := "/usr/share/home-lab-setup-scripts/scripts/wireguard/generate-keys.sh"
    output, err := shell.RunScript(scriptPath, s.config.Get("WG_CONFIG_DIR"))
    if err != nil {
        return fmt.Errorf("key generation failed: %w", err)
    }
    s.ui.Success("Keys generated successfully")
    return nil
}
```

## CLI Interface Design

### Main Menu (Interactive Mode)

```
$ homelab-setup

╔════════════════════════════════════════════════════════════╗
║         UBlue uCore Homelab Setup                          ║
╚════════════════════════════════════════════════════════════╝

Setup Options:
  [A] Run All Steps (Complete Setup)
  [Q] Quick Setup (Skip WireGuard)

Individual Steps:
  [0] ✓ Pre-flight Check
  [1] ✓ User Setup
  [2]   Directory Setup
  [3]   WireGuard Setup (optional)
  [4]   NFS Setup
  [5]   Container Setup
  [6]   Service Deployment

Other Options:
  [T] Troubleshooting Tool
  [S] Show Setup Status
  [R] Reset Setup
  [H] Help
  [X] Exit

Enter your choice:
```

### Command-Line Mode

```bash
# Run all steps
homelab-setup run all

# Run specific step
homelab-setup run preflight
homelab-setup run user
homelab-setup run directory

# Quick setup (skip optional steps)
homelab-setup run quick

# Check status
homelab-setup status

# Reset markers
homelab-setup reset

# Troubleshoot
homelab-setup troubleshoot

# Show version
homelab-setup version
```

### Non-Interactive Mode (for automation)

```bash
# Pass config via flags
homelab-setup run all \
  --non-interactive \
  --setup-user=containeruser \
  --nfs-server=192.168.7.10 \
  --skip-wireguard

# Read config from file
homelab-setup run all --config=/path/to/config.yaml
```

## Configuration Management

### Configuration File Format

**Location**: `~/.homelab-setup.conf` (keep existing format for compatibility)

**Format**: Key=Value (existing bash format)

```ini
CONTAINER_RUNTIME=podman
SETUP_USER=containeruser
PUID=1001
PGID=1001
TZ=America/Chicago
NFS_SERVER=192.168.7.10
CONTAINERS_BASE=/srv/containers
APPDATA_BASE=/var/lib/containers/appdata
```

### Go Config Loading

```go
type Config struct {
    filePath string
    data     map[string]string
}

func LoadConfig(path string) (*Config, error) {
    // Read file
    // Parse key=value lines
    // Return populated config
}

func (c *Config) Get(key string) (string, error) {
    // Return value or error if not found
}

func (c *Config) Set(key string, value string) error {
    c.data[key] = value
    return c.Save()
}
```

### Marker Files

**Location**: `~/.local/homelab-setup/`

**Files**:
- `preflight-complete`
- `user-setup-complete`
- `directory-setup-complete`
- `wireguard-setup-complete`
- `nfs-setup-complete`
- `container-setup-complete`
- `service-deployment-complete`

```go
type Markers struct {
    dir string
}

func (m *Markers) Create(name string) error {
    // Create marker file
}

func (m *Markers) Exists(name string) bool {
    // Check if marker file exists
}

func (m *Markers) Remove(name string) error {
    // Delete marker file
}
```

## Error Handling Strategy

### Go Error Handling

```go
// Wrap errors with context
if err := step.Run(ctx); err != nil {
    return fmt.Errorf("step %s failed: %w", step.Name(), err)
}

// Custom error types
type StepError struct {
    Step    string
    Cause   error
    Message string
}

func (e *StepError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Step, e.Message, e.Cause)
}

// Recoverable vs fatal errors
type ErrorSeverity int

const (
    SeverityWarning ErrorSeverity = iota  // Continue execution
    SeverityError                          // Stop step, allow retry
    SeverityFatal                          // Stop entire setup
)
```

### User-Friendly Error Messages

```go
func (s *PreflightStep) Run(ctx context.Context) error {
    if !s.system.PackageInstalled("nfs-utils") {
        return &StepError{
            Step:     "preflight",
            Message:  "Required package 'nfs-utils' is not installed",
            Cause:    nil,
            Solution: "Run: sudo rpm-ostree install nfs-utils && sudo systemctl reboot",
        }
    }
    return nil
}
```

## Testing Strategy

### Unit Tests

Each package should have comprehensive unit tests:

```go
// internal/common/validation_test.go
func TestValidateIP(t *testing.T) {
    tests := []struct {
        input   string
        isValid bool
    }{
        {"192.168.1.1", true},
        {"256.1.1.1", false},
        {"not-an-ip", false},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            err := ValidateIP(tt.input)
            if tt.isValid && err != nil {
                t.Errorf("expected valid IP, got error: %v", err)
            }
            if !tt.isValid && err == nil {
                t.Errorf("expected invalid IP, got no error")
            }
        })
    }
}
```

### Integration Tests

Test full step execution in isolated environment:

```go
// internal/steps/user_test.go
func TestUserSetupStep(t *testing.T) {
    // Create temp config
    // Mock UI responses
    // Run step
    // Verify config changes
}
```

### Mock UI for Testing

```go
type MockUI struct {
    responses map[string]interface{}
}

func (m *MockUI) PromptYesNo(prompt string, defaultYes bool) (bool, error) {
    if resp, ok := m.responses[prompt].(bool); ok {
        return resp, nil
    }
    return defaultYes, nil
}
```

## Build and Distribution

### Makefile

```makefile
.PHONY: build install test clean

build:
	go build -o bin/homelab-setup ./cmd/homelab-setup

install:
	go install ./cmd/homelab-setup

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -rf bin/
	rm -f coverage.out

# Build for multiple architectures
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/homelab-setup-linux-amd64 ./cmd/homelab-setup
	GOOS=linux GOARCH=arm64 go build -o bin/homelab-setup-linux-arm64 ./cmd/homelab-setup
```

### Installation in BlueBuild Image

Update `files/` directory structure:

```
files/
├── scripts/
│   └── install-homelab-setup-binary.sh    # Download/install Go binary
└── system/
    ├── usr/
    │   ├── bin/
    │   │   └── homelab-setup                # Go binary
    │   └── share/
    │       └── home-lab-setup-scripts/
    │           ├── scripts/                  # Legacy bash scripts (if needed)
    │           │   ├── wireguard/
    │           │   │   └── generate-keys.sh
    │           │   └── ...
    │           └── templates/
    │               ├── compose-setup/
    │               └── wireguard-setup/
    └── etc/
        └── systemd/
            └── system/
                └── homelab-setup.service     # Optional systemd service
```

## Migration Path

### Phase 1: Foundation (Week 1)
- [ ] Set up Go project structure
- [ ] Implement config package
- [ ] Implement UI package (prompts, output)
- [ ] Implement common validators
- [ ] Write unit tests

### Phase 2: System Operations (Week 2)
- [ ] Implement system package
  - [ ] Package detection
  - [ ] Service management
  - [ ] User/group operations
  - [ ] File system operations
  - [ ] Network tests
- [ ] Write integration tests

### Phase 3: Simple Steps (Week 3)
- [ ] Implement preflight step
- [ ] Implement user setup step
- [ ] Implement directory setup step
- [ ] Implement NFS setup step
- [ ] Test each step individually

### Phase 4: Complex Steps (Week 4)
- [ ] Implement container setup step
- [ ] Implement service deployment step
- [ ] Implement WireGuard step (with shell-out for keygen)
- [ ] Implement troubleshooting tool

### Phase 5: CLI & Polish (Week 5)
- [ ] Implement main CLI with cobra
- [ ] Add interactive menu
- [ ] Add command-line flags
- [ ] Add non-interactive mode
- [ ] Write comprehensive documentation

### Phase 6: Testing & Deployment (Week 6)
- [ ] End-to-end testing
- [ ] Build for multiple architectures
- [ ] Update BlueBuild image config
- [ ] Create installation docs
- [ ] Deprecate old bash scripts

## Benefits Summary

### For Users
✅ **Faster startup** - Compiled binary vs bash script parsing
✅ **Better error messages** - Structured errors with solutions
✅ **No missing dependencies** - Single binary, no Python/jq needed
✅ **Consistent behavior** - No bash version differences
✅ **Better progress feedback** - Native progress bars

### For Maintainers
✅ **Type safety** - Catch errors at compile time
✅ **Better IDE support** - Go tooling is excellent
✅ **Easier testing** - Native test framework
✅ **Easier debugging** - Better stack traces
✅ **Code reuse** - Packages and interfaces
✅ **Better documentation** - Godoc comments

### For the Project
✅ **Reduced complexity** - ~4600 lines bash → ~3000-4000 lines Go (estimated)
✅ **Better security** - Less shell injection risk
✅ **Cross-platform potential** - Can support other immutable distros
✅ **Modern toolchain** - Leverages Go ecosystem

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Go learning curve for contributors | Medium | Good documentation, simple architecture |
| Loss of bash script flexibility | Low | Keep shell-out option for complex operations |
| Breaking changes for users | Medium | Maintain config file format, migration guide |
| Increased binary size | Low | Single ~10MB binary vs scattered scripts |
| Complex systemd operations in Go | Medium | Use go-systemd library OR shell out to systemctl |

## Conclusion

Rewriting the homelab setup scripts in Go will significantly improve maintainability, reliability, and user experience. The hybrid approach allows us to leverage Go's strengths while keeping complex operations in bash where appropriate.

**Recommended Next Steps:**
1. Review and approve this plan
2. Create a feature branch: `feature/go-rewrite`
3. Start with Phase 1 (Foundation)
4. Iterate and test each phase
5. Deploy alongside bash scripts initially
6. Deprecate bash scripts after stabilization period
