# Ad Event Processor

High-performance Go backend for real-time, distributed ad event processing.

## Features
- **Durable Ingestion**: Redis Streams for high-throughput, loss-resistant event hand-off.
- **Atomic Aggregation**: Database-level exactly-once aggregation via SQL CTEs.
- **Intelligent Filtering**: Distributed IP rate limiting and deduplication middleware.
- **Stateless Scaling**: Horizontal scalability with zero in-memory state in workers.
- **Observability**: Built-in Prometheus metrics, Grafana dashboards, and pprof profiling.

## Docs
- [**Architecture**](docs/architecture.md) - Internal logic & distributed lifecycle.
- [**Development**](docs/development.md) - Setup, Testing & Tooling.

## Quick Start
```bash
# Start infrastructure (Postgres + Redis)
docker compose up -d

# Run the test suite
make test

# Start the server
go run cmd/server/main.go
```

## Monitoring
- Prometheus: `:9095` (customizable)
- Grafana: `:3005` (admin/admin)
- Health: `GET /health`
- Metrics: `GET /metrics`
