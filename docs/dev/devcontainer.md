# VS Code Dev Container for UBlue uCore Homelab Setup

This directory contains the configuration for a VS Code development container that provides a consistent, reproducible development environment for the homelab-setup Go project.

## Features

- **Go 1.24** (Debian Bookworm base)
- **Pre-installed Go tools**:
  - `gopls` - Go language server
  - `dlv` - Delve debugger
  - `staticcheck` - Static analysis
  - `golangci-lint` - Linter aggregator
- **System tools** for testing:
  - systemd
  - rpm
  - podman
  - Network utilities (ping, dig, etc.)
- **VS Code extensions**:
  - Go extension
  - Makefile tools
  - YAML support
  - GitLens
  - Code spell checker
- **Non-root user** (`vscode`) with passwordless sudo
- **Docker socket mounted** for container operations

## Prerequisites

1. **VS Code** with the **Remote - Containers** extension installed
2. **Docker** or **Podman** running on your host machine

## Usage

### Opening the Project in the Dev Container

1. Open VS Code
2. Open the `homelab-coreos-minipc` directory
3. When prompted, click **"Reopen in Container"**
   - Or use the command palette (F1) and select: **"Remote-Containers: Reopen in Container"**

VS Code will:
- Build the container image (first time only)
- Start the container
- Install VS Code extensions
- Run post-create commands to set up Go tools

### Manual Container Build

If you want to pre-build the container image:

```bash
cd /path/to/homelab-coreos-minipc/.devcontainer
docker build -t homelab-setup-devcontainer .
```

### Working in the Container

Once inside the container:

```bash
# Navigate to the Go project
cd homelab-setup

# Download dependencies
go mod download

# Run tests
go test ./...

# Build the binary
make build

# Run the binary
./bin/homelab-setup --help
```

## Customization

### Adding VS Code Extensions

Edit `.devcontainer/devcontainer.json` and add extension IDs to the `extensions` array:

```json
"extensions": [
    "golang.go",
    "your.extension-id"
]
```

### Installing Additional Tools

Edit `.devcontainer/Dockerfile` and add packages to the `apt-get install` command:

```dockerfile
RUN apt-get update && apt-get install -y \
    your-package-name \
    && apt-get clean
```

### Environment Variables

Edit `.devcontainer/devcontainer.json` and modify the `remoteEnv` section:

```json
"remoteEnv": {
    "MY_VAR": "value"
}
```

## Troubleshooting

### Container fails to start

1. Check Docker/Podman is running: `docker ps` or `podman ps`
2. Check the VS Code Output panel for errors
3. Try rebuilding: **"Remote-Containers: Rebuild Container"**

### Go tools not working

Run the post-create command manually:

```bash
cd homelab-setup
go mod download
go install golang.org/x/tools/gopls@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Permission issues

The dev container runs as the `vscode` user (UID 1000). If you have permission issues:

```bash
# Fix ownership of the Go cache
sudo chown -R vscode:vscode /home/vscode/go

# Fix ownership of workspace files
sudo chown -R vscode:vscode /workspace
```

## Performance Tips

- **Use Docker volumes** instead of bind mounts for better I/O performance on macOS/Windows
- **Exclude node_modules and build artifacts** from file watching
- **Close unused files** in the editor to reduce memory usage

## Security Notes

- The container runs with `SYS_PTRACE` capability for debugging
- The Docker socket is mounted for container operations (use with caution)
- The `vscode` user has passwordless sudo (development environment only)
- **Do not use this container for production workloads**

## References

- [VS Code Dev Containers Documentation](https://code.visualstudio.com/docs/devcontainers/containers)
- [Dev Container Specification](https://containers.dev/)
- [Go in VS Code](https://code.visualstudio.com/docs/languages/go)
