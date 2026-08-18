# BidShard Dev Container

This project includes a VS Code Dev Container configuration to provide a consistent development environment across different operating systems.

## Why use the Dev Container?

- **Linux-only features**: BidShard requires a Linux environment for eBPF/XDP development and certain performance tests.
- **Pre-installed dependencies**: The container comes pre-configured with:
    - Go 1.25
    - Clang/LLVM (for BPF compilation)
    - Node.js & NPM (for admin UI)
    - PostgreSQL & Redis CLI tools
    - `sqlc`, `buf`, and `task` (task runner)
- **Docker-in-Docker**: You can run the full BidShard stack using `docker-compose` directly inside the container.

## Prerequisites

1.  **Visual Studio Code**
2.  **Dev Containers extension** for VS Code
3.  **Docker Desktop** (or Docker on Linux)

## Getting Started

1.  Open the project folder in VS Code.
2.  When prompted, click **"Reopen in Container"**.
3.  Wait for the build to finish (the first build might take a few minutes).
4.  The container will automatically download Go modules, install NPM packages, and run initial code generation.

## Usage

Use the integrated terminal to run project tasks:

- `task gen`: Run all code generation (SQL, Proto, BPF).
- `task up`: Start the infrastructure stack.
- `task test`: Run fast tests.
- `task --list`: See all available tasks.

## Troubleshooting

- **eBPF Attach Failures**: If your host kernel version is lower than 5.8, some BPF features (like XDP or tracepoints) might fail to attach even inside the container.
- **Docker Performance**: On macOS/Windows, ensure Docker is allocated at least 4GB of RAM for the full stack.
