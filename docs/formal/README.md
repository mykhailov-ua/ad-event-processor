# Formal verification artifacts (optional M3.1 / M3.2)

Not run in default CI. Use for design reviews before state-machine or activation-policy changes.

| File | Tool | Property |
| --- | --- | --- |
| `licensing_state.tla` | TLC | P-C3 ingest vs state; monotonic expiry sketch |
| `licensing_state.cfg` | TLC | Invariants for `licensing_state.tla` |
| `licensing_activation.als` | Alloy 6 | `max_activations` cap |

```bash
# Optional local runs (tools not vendored in repo)
tlc licensing_state.tla -config licensing_state.cfg
alloy6 exec licensing_activation.als
```

Primary automated verification: `make license-verify` (`internal/licensing/VERIFY.md`).
