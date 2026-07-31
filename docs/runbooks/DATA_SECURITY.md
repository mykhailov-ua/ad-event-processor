# Runbook: data security on self-hosted installs

How operators protect ad event data, credentials, and backups on **their own bare metal**. eSPX provides application-level controls; **encryption at rest and network perimeter** are operator responsibilities.

**Protection overview (vendor IP + operator data + trust):** [PROTECTION.md](../PROTECTION.md).

Related: [SELF_HOSTED.md](../SELF_HOSTED.md), [TELEMETRY_AND_TRUST.md](../TELEMETRY_AND_TRUST.md), [LICENSE_COMMERCE.md](../LICENSE_COMMERCE.md), [ARCHITECTURE.md](../ARCHITECTURE.md), [.cursor/COMPLIANCE_MATRIX.md](../../.cursor/COMPLIANCE_MATRIX.md).

---

## Threat model

| Threat | Example | Primary mitigation |
| :--- | :--- | :--- |
| **Network** | Sniffing `/track`, admin API, Redis/PG on LAN | TLS, private VLAN, firewall |
| **Disk / backup** | Stolen VPS, `pg_dump`, CH spool mmap | LUKS, encrypted backups |
| **Insider** | Admin, contractor with SSH | RBAC, audit log, least privilege |
| **Process compromise** | RCE in management, leaked `.env` | Vault, non-root, segmentation |
| **Vendor egress** | Fear of traffic-bundle leak via binary | Telemetry opt-in off, schema audit, allowlist — TELEMETRY_AND_TRUST.md |

Self-hosted: **keys and plaintext policy belong to the operator**. The vendor ships mechanisms and this runbook; the operator owns the threat model for their jurisdiction and traffic.

---

## What eSPX provides (in product)

### Perimeter and transport

| Control | Where |
| :--- | :--- |
| TLS termination at edge | Nginx/OpenResty (HTTPS/H2/H3 → tracker H1.1) |
| Redis authentication | `requirepass` / ACL (compose reference) |
| Operator auth | Argon2id passwords, PASETO sessions (`internal/auth`) |
| Control-plane HMAC | `TCPControlHMACSecret` (`internal/management/tcp_control_server.go`) |
| Edge drop / rate limit | XDP, nginx Lua, optional tarpit (`CMP-DEF-*` in compliance matrix) |

### Data minimization and pseudonymization

| Control | Where |
| :--- | :--- |
| PII hashing before ClickHouse insert | `pkg/piihash` — HighwayHash + `PII_SALT_VERSION` (GAP-DATA-01) |
| Raw `ip_address` / `user_agent` dropped in CH | Migration `00010_pii_hash_columns.sql` |
| Fraud telemetry rings | Lossy buffers; not durable PII store |

### Access and audit

| Control | Where |
| :--- | :--- |
| RBAC on management API | Permission strings (`campaigns:read`, …) |
| Admin audit trail | `admin_audit_log` + outbox for blacklist mutations |
| Tenant isolation | `ensureCampaignAccess` for advertiser API keys |

### What the product does **not** provide by default

| Gap | Current state |
| :--- | :--- |
| Encryption at rest (disk) | Operator OS / volume encryption |
| PostgreSQL `events.ip_address`, `user_agent` | **Plaintext** (settlement, dedup, audit) |
| Redis values | In-memory / RDB/AOF without app-level encryption |
| mTLS between all internal services | Reference compose is dev-friendly; prod is operator-owned |
| Central secrets manager | `.env` / k8s secrets in reference deploy |

---

## Defense in depth (operator checklist)

```text
[1] Network perimeter   — firewall; no public PG/Redis/CH
[2] TLS in transit      — edge + recommended internal TLS
[3] At-rest encryption  — LUKS / encrypted backups
[4] App minimization    — CH hashes; PG retention policy
[5] Secrets             — Vault; unique salts; rotation
[6] Audit               — RBAC, admin_audit_log, egress logs
```

### Minimum viable security (sell sheet / install gate)

- [ ] LUKS (or cloud volume encryption) on PG, Redis, CH, and `CHSpoolDir` volumes
- [ ] Firewall: only `443` / `8180` (or chosen ingress) public; data stores on private subnet
- [ ] `DB_DSN` with `sslmode=verify-full` and server CA pinned
- [ ] Unique `PII_SALT_HEX` (32 random bytes, hex-encoded) — not copied from `.env.example`
- [ ] Strong `REDIS_PASSWORD`; Redis not bound to `0.0.0.0` on untrusted interfaces
- [ ] `ESPX_TELEMETRY_OPT_IN=0` unless policy reviewed ([TELEMETRY_AND_TRUST.md](../TELEMETRY_AND_TRUST.md))
- [ ] RBAC: operator admins ≠ advertiser API keys
- [ ] Encrypted backups with offline key
- [ ] Documented PG `events` retention (see § PostgreSQL below)

---

## Layer 1 — Network segmentation

| Service | Exposure |
| :--- | :--- |
| Nginx / edge | Internet or CDN-facing |
| Tracker pool | Internal LB only |
| Redis ×4 | Private network; Sentinel if used |
| PostgreSQL | Private network only |
| ClickHouse | Private network only |
| management / gRPC | Admin VPN or bastion; not public unless intentional |

License server (vendor): optional second egress allowlist entry — separate from telemetry host if used.

---

## Layer 2 — Encryption in transit

| Channel | Recommendation |
| :--- | :--- |
| Users → edge | TLS 1.2+ (required) |
| Edge → tracker | TLS or isolated L2/L3 network |
| Apps → PostgreSQL | `sslmode=verify-full` in `DB_DSN` |
| Apps → Redis | Redis 6+ TLS + ACL (password alone is baseline) |
| Apps → ClickHouse | Native protocol over TLS |
| gRPC (auth, payment, billing) | mTLS or single trusted VLAN |

Reference `docker-compose.yaml` may use plaintext on the bridge for local dev. **Production profile must document TLS** (GAP-DATA-02).

Example DSN fragment:

```text
postgres://user:pass@pg.internal:5432/espx?sslmode=verify-full&sslrootcert=/etc/espx/ca.pem
```

---

## Layer 3 — Encryption at rest

| Store | Operator action |
| :--- | :--- |
| PostgreSQL data directory | LUKS or provider volume encryption; optional PG TDE if licensed |
| ClickHouse data | Encrypted volume; raw IP columns removed in app layer |
| Redis RDB/AOF | Encrypted volume; avoid storing unnecessary PII in keys |
| CH mmap spool (`CHSpoolDir`) | Same encrypted volume as CH or dedicated encrypted mount |
| Backups (`pg_dump`, CH snapshots) | GPG, age, or KMS — keys not on same host as data |

eSPX does not encrypt database files inside the application.

---

## Layer 4 — Application data handling

### ClickHouse (analytics)

- Inserts use `ip_hash`, `ua_hash`, `user_id_hash` via `pkg/piihash`.
- Configure before production:

```bash
# .env — 64 hex chars = 32 bytes
PII_SALT_HEX=<generate-with-openssl-rand-hex-32>
PII_SALT_VERSION=1
```

Fallback derives salt from `TOKEN_SYMMETRIC_KEY` if `PII_SALT_HEX` unset — **do not rely on fallback in production**.

**Salt rotation:** increment `PII_SALT_VERSION`; new rows use new version; old hashes are not reversible across versions (by design).

### PostgreSQL (financial + events)

| Table / column | Sensitivity | Policy |
| :--- | :--- | :--- |
| `events.ip_address`, `user_agent` | High | Plaintext by default; `EVENTS_HASH_IP_AT_INSERT=1` stores `ip_hash` only in new rows |
| `balance_ledger`, `campaigns` | Financial | Encrypted disk + strict DB roles |
| `auth` users | Credentials | Argon2id hashes only |

**Recommended operator policy:**

- `EventsRetentionWorker` (management): batched delete of `events` older than `EVENTS_RETENTION_DAYS` (default 90).
- Optional: partition by `created_date` and detach/drop old partitions.
- Do not expose PG port publicly; read-only role for reporting tools.

**Hash-at-insert (`EVENTS_HASH_IP_AT_INSERT=1`):** processor writes `ip_hash` (HighwayHash + `PII_SALT_HEX`) and leaves `ip_address` empty. Tracker/redis dedup, Lua filters, and settlement idempotency still use raw IP on the hot path before PG insert; PG reporting must join on `ip_hash`, not plaintext IP.

### Redis (hot path)

- Budget keys, dedup, streams — performance-critical.
- **Do not log Redis values** in application debug.
- TTL on dedup/fcap keys (Lua design) limits exposure window.
- Treat RDB snapshots as sensitive; encrypt backup media.

---

## Layer 5 — Secrets management

| Secret | Storage | Notes |
| :--- | :--- | :--- |
| `PII_SALT_HEX` | Vault / sealed secret | Unique per deployment |
| `TOKEN_SYMMETRIC_KEY` | Vault | PASETO; separate from PII salt |
| `DB_DSN`, `PAYMENT_DB_DSN` | Vault | Rotate DB passwords independently |
| `REDIS_PASSWORD` | Vault | Per-shard same or distinct per policy |
| `STRIPE_*`, crypto webhook secrets | Vault | Operator treasury (Layer O) |
| `license.jwt` | File mode `0600` | Dedicated service user |
| `SETTLEMENT_INTERNAL_TOKEN`, gRPC tokens | Vault | Internal mesh only |

Never commit production secrets to git. `.env.example` contains placeholders only.

---

## Layer 6 — Access control and audit

| Practice | Detail |
| :--- | :--- |
| Separate DB roles | Processor write; `CHQuery` readonly; admin migrations via migration user |
| Management RBAC | Least privilege per operator staff |
| Advertiser API keys | Scoped to single `customer_id` |
| `admin_audit_log` | Review on blacklist and config mutations |
| SSH | Bastion; no shared root |
| Egress monitoring | Log outbound to license (and optional telemetry) hosts |

---

## Encryption vs vendor trust

| Protection | Protects against |
| :--- | :--- |
| LUKS / backup encryption | Host theft, snapshot leak, third-party hoster |
| TLS | Network adversary on path |
| PII hash in CH | Analytics DB dump |
| Telemetry opt-in off + allowlist | Vendor competitive intelligence concerns |

Disk encryption **does not** prove the vendor binary has no backdoor. Trust model for egress: [TELEMETRY_AND_TRUST.md](../TELEMETRY_AND_TRUST.md). Paranoid profile: air-gap license file, telemetry off, egress firewall deny-by-default.

---

## Open product decisions (architecture)

Tracked in backlog as **GAP-DATA-02** and related items:

| # | Topic | Status |
| :---: | :--- | :--- |
| 1 | PG `ip_address`: retention-only vs hash-at-insert | Open |
| 2 | Production compose profile with internal TLS defaults | Open |
| 3 | Redis TLS in reference production profile | Open |
| 4 | Envelope encryption for `events.payload` (operator DEK) | Open |
| 5 | Installer checklist gate for MVSS items above | `espx doctor --checklist` (see deploy/installer/README.md) |
| 6 | CH spool segment encryption vs LUKS-only | Open |

---

## Incident response (operator)

1. Rotate `REDIS_PASSWORD`, DB passwords, `TOKEN_SYMMETRIC_KEY` (forces re-login).
2. Revoke API keys in management.
3. If PII salt compromised: set new `PII_SALT_HEX` + bump `PII_SALT_VERSION`; plan CH backfill policy (hashes not reversible).
4. Review `admin_audit_log` and egress logs.
5. Restore from encrypted backup if ransomware; verify backup key was offline.

---

## Related engineering gaps

| ID | Topic |
| :--- | :--- |
| GAP-DATA-01 | PII hashing before ClickHouse insert — **shipped** |
| GAP-DATA-02 | PG events retention, optional IP hashing, production TLS profile — **shipped** (P19) |

Open backlog: `.cursor/BACKLOG.md`.
