# Use the official Go image as base
# Match the Go version requirement from the project (1.23+)
FROM golang:1.24-bookworm

# Arguments for non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

# Install additional OS packages
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    # Development tools
    build-essential \
    git \
    curl \
    wget \
    vim \
    nano \
    sudo \
    # System tools that the Go app might interact with
    systemd \
    rpm \
    # Networking tools for testing
    iproute2 \
    iputils-ping \
    net-tools \
    dnsutils \
    # Container runtimes (for testing container detection)
    podman \
    # Shell and utilities
    bash-completion \
    jq \
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    # Add sudo support for the non-root user
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Set up Go environment for the non-root user
ENV GOPATH=/home/$USERNAME/go
ENV PATH=$GOPATH/bin:$PATH

# Create Go workspace directories
RUN mkdir -p $GOPATH/src $GOPATH/bin $GOPATH/pkg \
    && chown -R $USERNAME:$USERNAME $GOPATH

# Install Go development tools (as root, they'll be accessible to all users)
RUN go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Set the working directory
WORKDIR /workspace

# Switch to non-root user
USER $USERNAME

# Set up bash completion and aliases
RUN echo 'alias ll="ls -alF"' >> ~/.bashrc \
    && echo 'alias la="ls -A"' >> ~/.bashrc \
    && echo 'alias l="ls -CF"' >> ~/.bashrc \
    && echo 'export PS1="\[\033[01;32m\]\u@devcontainer\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> ~/.bashrc

# Configure git to be safe with the workspace directory
RUN git config --global --add safe.directory /workspace

# Default command
CMD ["/bin/bash"]
