# Ad Event Processor

High-throughput Go backend for real-time ad event processing.

## Features
- Worker-pool batching (`COPY` protocol).
- Sequential graceful shutdown (no data loss).
- Atomic in-memory aggregation.
- Prometheus & Grafana integration.

## Docs
- [**Architecture**](docs/architecture.md) - Internal logic & lifecycle.
- [**Development**](docs/development.md) - Build & Test setup.

## Quick Start
```bash
# Start infrastructure
docker compose up -d

# Run tests
make test
```

## Monitoring
- Prometheus: `:9090`
- Grafana: `:3000` (admin/admin)
