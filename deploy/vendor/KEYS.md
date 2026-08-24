# Ed25519 license keys

Per-cohort public keys for offline JWT verification on customer installs.

- Default pilot key: [license_public.key](./license_public.key) (`kid`: `2026-01`).
- Layout: `keys/<kid>/license_public.key` (private keys local only, gitignored).

Issue: `go run ./cmd/license-issue`. Verify catalog: `make license-verify`.
