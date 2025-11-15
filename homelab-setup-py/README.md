# UBlue uCore Homelab Setup (Python Edition)

Modern, maintainable Python scripts for setting up a UBlue uCore homelab environment.

## Features

- **Interactive Setup**: Guided configuration with rich terminal UI
- **Modular Design**: Each component (WireGuard, NFS, containers) can be configured independently
- **Comprehensive Validation**: Preflight checks ensure system readiness
- **Container Management**: Support for Podman and Docker with automatic detection
- **Troubleshooting**: Built-in diagnostics and health checks

## Requirements

- Python 3.9 or higher
- UBlue uCore (rpm-ostree based system)
- Sudo privileges

## Installation

```bash
# Install in development mode
cd homelab-setup-py
pip install -e .

# Or install with development dependencies
pip install -e ".[dev]"
```

## Usage

```bash
# Run all setup steps interactively
homelab-setup run-all

# Run individual steps
homelab-setup preflight          # Pre-flight checks
homelab-setup user               # User setup
homelab-setup directories        # Directory creation
homelab-setup wireguard          # WireGuard VPN
homelab-setup nfs                # NFS mounts
homelab-setup containers         # Container setup
homelab-setup deploy             # Deploy services

# Troubleshooting
homelab-setup troubleshoot       # Run diagnostics
homelab-setup troubleshoot --all # Full diagnostic report
```

## Development

```bash
# Run tests
pytest

# Type checking
mypy homelab_setup

# Code formatting
black homelab_setup
ruff check homelab_setup
```

## Migration from Bash Scripts

This Python rewrite maintains feature parity with the original bash scripts while providing:
- Better error handling and validation
- Improved code maintainability
- Type safety
- Unit testability
- Rich terminal UI with progress indicators

The original bash scripts remain in `files/system/usr/share/home-lab-setup-scripts/scripts/` for reference.
