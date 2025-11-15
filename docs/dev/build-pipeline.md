# Automated Binary Builds

## Overview

The `homelab-setup` binary is automatically built and committed to the repository whenever changes are made to the Go code in the `homelab-setup/` directory.

## GitHub Actions Workflow

The workflow is defined in `.github/workflows/build-homelab-setup.yml` and:

1. **Triggers** when:
   - Changes are pushed to `homelab-setup/**` on the `main` branch
   - Pull requests modify files in `homelab-setup/**`
   - Manually dispatched via workflow_dispatch

2. **Build Process**:
   - Sets up Go 1.23
   - Runs all unit tests (`go test ./... -v`)
   - Builds the binary using `make build`
   - Copies the binary to `files/system/usr/local/bin/homelab-setup`
   - Verifies the binary works by running `homelab-setup version`

3. **Automatic Commit**:
   - If the binary has changed, the workflow commits it back to the repository
   - Commit message: `chore: rebuild homelab-setup binary [skip ci]`
   - The `[skip ci]` tag prevents infinite build loops

4. **Artifact Upload**:
   - The binary is also uploaded as a GitHub Actions artifact
   - Retained for 30 days for easy download

## Binary Location

The compiled binary is committed to:
```
files/system/usr/local/bin/homelab-setup
```

This path is chosen because:
- It matches the BlueBuild image structure
- `/usr/local/bin` is the standard location for user-installed binaries
- It's automatically included in the container image builds

## Local Development

During local development:

```bash
# Build locally
cd homelab-setup
make build

# The local binary is in:
# homelab-setup/bin/homelab-setup

# To test without installing:
./bin/homelab-setup version

# To install locally:
sudo cp bin/homelab-setup /usr/local/bin/
```

Local build artifacts in `homelab-setup/bin/` are excluded by `.gitignore` to prevent accidentally committing unintended versions.

## Manual Build and Commit

If you need to manually rebuild and commit the binary:

```bash
# Build the binary
cd homelab-setup
make build

# Copy to the committed location
mkdir -p ../files/system/usr/local/bin
cp bin/homelab-setup ../files/system/usr/local/bin/

# Verify
../files/system/usr/local/bin/homelab-setup version

# Commit (if desired)
git add ../files/system/usr/local/bin/homelab-setup
git commit -m "chore: rebuild homelab-setup binary"
git push
```

## Workflow Permissions

The workflow uses the built-in `GITHUB_TOKEN` which has:
- Read access to repository contents
- Write access to commit changes back to the repository

No additional secrets or tokens are required.

## Debugging Failed Builds

If the workflow fails:

1. Check the Actions tab in GitHub for detailed logs
2. Common issues:
   - Test failures: Fix tests and push again
   - Build errors: Check Go version compatibility
   - Permission errors: Ensure workflow has write permissions

To test locally before pushing:

```bash
cd homelab-setup

# Run tests
make test

# Build
make build

# Verify
./bin/homelab-setup version
```
