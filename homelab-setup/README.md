# Homelab Setup - Go Implementation

A comprehensive setup tool for configuring homelab services on UBlue uCore, rewritten in Go for better maintainability, reliability, and ease of use.

## Features

- **Single compiled binary** - No runtime dependencies (Python, jq, etc.)
- **Type-safe** - Compile-time error checking
- **Well-tested** - Comprehensive unit tests
- **Interactive UI** - User-friendly prompts and colored output
- **Command-line mode** - Scriptable for automation

## Installation

### From Source

```bash
make build
sudo cp bin/homelab-setup /usr/local/bin/
```

### Build for Different Architectures

```bash
make build-all
```

## Usage

### Interactive Menu

Run without arguments to start the interactive menu:

```bash
homelab-setup
```

### Command-Line Mode

```bash
# Show version
homelab-setup version

# Run specific steps (coming in Phase 2)
homelab-setup run preflight
homelab-setup run user
homelab-setup run all

# Check status
homelab-setup status

# Troubleshoot
homelab-setup troubleshoot
```

## Development

### Prerequisites

- Go 1.21 or higher
- make

### Building

```bash
# Build binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```

### Project Structure

```
homelab-setup/
├── cmd/
│   └── homelab-setup/          # CLI entry point
├── internal/
│   ├── config/                 # Configuration management
│   ├── ui/                     # User interface (prompts, output)
│   ├── common/                 # Validators and utilities
│   ├── system/                 # System operations
│   ├── steps/                  # Setup steps
│   └── troubleshoot/           # Diagnostic tools
├── pkg/
│   └── version/                # Version information
├── Makefile                    # Build automation
└── README.md
```

### Testing

Run all tests:

```bash
make test
```

Run tests with coverage report:

```bash
make test-coverage
```

View coverage in browser:

```bash
make test-coverage
# Open coverage.html in browser
```

## Implementation Status

### Phase 1: Foundation ✅

- [x] Project structure
- [x] Configuration management (config files, markers)
- [x] UI package (prompts, colored output)
- [x] Common validators (IP, port, path, username, domain, timezone)
- [x] Unit tests
- [x] Build system (Makefile)
- [x] Version command

### Phase 2: System Operations (Next)

- [ ] Package detection
- [ ] Service management (systemd)
- [ ] User/group operations
- [ ] File system operations
- [ ] Network tests
- [ ] Container runtime detection

### Phase 3: Setup Steps

- [ ] Preflight checks
- [ ] User setup
- [ ] Directory setup
- [ ] NFS setup
- [ ] Container setup
- [ ] Service deployment
- [ ] WireGuard setup

### Phase 4: CLI & Features

- [ ] Interactive main menu
- [ ] Run command (individual steps)
- [ ] Status command
- [ ] Reset command
- [ ] Troubleshooting tool
- [ ] Non-interactive mode

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [survey](https://github.com/AlecAivazis/survey) - Interactive prompts
- [color](https://github.com/fatih/color) - Colored output

## Configuration

Configuration is stored in `~/.homelab-setup.conf` (same format as bash version):

```ini
CONTAINER_RUNTIME=podman
SETUP_USER=containeruser
PUID=1001
PGID=1001
TZ=America/Chicago
NFS_SERVER=192.168.7.10
```

Completion markers are stored in `~/.local/homelab-setup/`:

```
preflight-complete
user-setup-complete
directory-setup-complete
nfs-setup-complete
container-setup-complete
service-deployment-complete
```

## License

See LICENSE file in the repository root.

## Contributing

See the implementation plan in `docs/go-rewrite-plan.md` for architecture details and roadmap.
