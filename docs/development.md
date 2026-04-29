# Development Guide

Tooling, testing, and maintenance workflow.

## Requirements
- **Go 1.25+**
- **Docker & Docker Compose**

## Makefile Targets

| Target | Action |
| :--- | :--- |
| `make fmt` | Format code via `gofmt`. |
| `make lint` | Run `golangci-lint` with `.golangci.yml`. |
| `make test` | Run all tests (Unit + Integration). |
| `make test-unit` | Run fast unit tests. |
| `make test-int` | Run integration tests (requires Docker). |
| `make build` | Build production Docker image. |

## Local Infrastructure
Spin up full stack:
```bash
docker compose up -d
```

## Testing

### Unit
- `tests/unit/`
- Isolated logic testing using Mocks.

### Integration
- `tests/integration/`
- Real PG interaction via **Testcontainers**.
- Fresh Postgres instance for each test suite.

## CI/CD
GitHub Actions workflow:
1. **Lint**: Style and static analysis.
2. **Test**: Full suite (Unit + Integration).
3. **Build**: Docker build validation.

## Metrics & UI
- **Prometheus**: `http://localhost:9090`
- **Grafana**: `http://localhost:3000` (admin/admin)
