FROM mcr.microsoft.com/devcontainers/go:1-1.25-bookworm

# Install additional tools
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    clang \
    llvm \
    libelf-dev \
    gcc-multilib \
    linux-headers-amd64 \
    bpftool \
    pkg-config \
    make \
    curl \
    git \
    nodejs \
    npm \
    docker.io \
    ripgrep \
    luajit \
    postgresql-client \
    redis-tools

# Install Go tools
RUN go install github.com/kyleconroy/sqlc/cmd/sqlc@latest \
    && go install github.com/bufbuild/buf/cmd/buf@latest \
    && go install github.com/go-task/task/v3/cmd/task@latest

# Set up workdir
WORKDIR /workspaces/bidshard
