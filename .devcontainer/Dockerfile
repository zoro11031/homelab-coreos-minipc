# Minimal Dockerfile for Go compilation only
FROM golang:1.24-bookworm

# Install ONLY what's needed to compile Go code
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    git \
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /workspace

# Default command
CMD ["/bin/bash"]
