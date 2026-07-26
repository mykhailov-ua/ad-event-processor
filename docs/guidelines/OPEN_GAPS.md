# Open Gaps

Open gaps only. Architecture: [../ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Priority

| P | Theme |
| :--- | :--- |
| P1 | Buyer UX / reporting |
| P3 | CH query governance, backlog observability |
| P4 | PII in CH, vendor telemetry |
| P5 | Crypto gateway, Postgres DR, multi-region game days |

---

## Gaps

| ID | Area | Notes |
| :--- | :--- | :--- |
| GAP-RTB-10 | RTB | Inventory expansion: placement/domain, creative-level auction, video/VAST |
| GAP-RTB-11 | RTB | Pre-auction caps: daypart bitmasks, frequency-cap pre-check |
| GAP-RTB-12 | RTB | Platform ops: CTV gtax, admin simulate, A/B cohorts, multi-region budget |
| GAP-OPS-03 | Operations | CH admin query governance; some paths bypass `CHQuery` |
| GAP-OPS-04 | Operations | DLQ/spool unified dashboard |
| GAP-PROD-01 | Product | Buyer/finance dashboards; scaffold routes return 501 |
| GAP-PROD-03 | Product | No OpenAPI; godoc only |
| GAP-GEO-01 | Geography | Multi-region game days not productized |
| GAP-GEO-02 | Geography | Postgres DR manual |
| GAP-PAY-01 | Payments | Crypto gateway |
| GAP-DATA-01 | Data | Raw PII in ClickHouse |
| GAP-CMP-01 | Compliance | Tarpit partial; compliance matrix open |
| GAP-ENG-01 | Engineering | Large flat `internal/management` |
| GAP-ENG-02 | Engineering | `cmd/broker` not in default compose |
| GAP-ENG-03 | Engineering | Vendor telemetry opt-in |
| GAP-DB-01 | Database | Logger group-commit fsync |
| GAP-DB-02 | Database | CH spool group-commit |
| GAP-DB-03 | Database | Weighted processor gates |

Suggested order: GAP-RTB-10..12 -> GAP-PROD-01 -> GAP-OPS-03/04 -> GAP-DATA-01 -> GAP-PAY-01 -> GAP-GEO-01/02.

Chaos catalog: [CHAOS.md](./CHAOS.md).
