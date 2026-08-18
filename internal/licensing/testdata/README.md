# Licensing golden vectors

Generated fixtures for `go test ./internal/licensing/`. Do not hand-edit without regenerating.

| File | Generator | Consumer |
| --- | --- | --- |
| `argon2id_hwid.json` | `WRITE_HWID_VECTORS=1 go test -run TestGenHWIDVectorArtifacts` | `TestHWID_GoldenVectors` |
| `mck_derivation.json` | `WRITE_MCK_VECTORS=1 go test -run TestGenMCKVectorArtifacts` | `TestDeriveMCK_GoldenVector` |
| `hkdf_rfc5869.json` | RFC 5869 cases 1–3 (HKDF-SHA-256 via `golang.org/x/crypto/hkdf`) | `TestHKDF_RFC5869Vectors` |

Regenerate HWID + MCK:

```bash
bash scripts/security/license_vector_gen.sh
```
