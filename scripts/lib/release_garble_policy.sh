#!/usr/bin/env bash

# Role: Library: Release garble policy helper.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/release_garble_policy.sh
release_garble_seed_ok() {
  if [[ "${RELEASE_GARBLE:-1}" != "1" ]]; then
    return 0
  fi
  if [[ -n "${GARBLE_SEED:-}" ]]; then
    return 0
  fi
  if [[ "${RELEASE_GARBLE_SKIP_SEED:-}" == "1" ]]; then
    echo "release_garble: GARBLE_SEED unset (RELEASE_GARBLE_SKIP_SEED=1 local dev only)" >&2
    return 0
  fi
  return 1
}

release_asset_seal_salt_ok() {
  if [[ "${RELEASE_GARBLE:-1}" != "1" && "${RELEASE_ASSET_SEAL_REQUIRED:-}" != "1" ]]; then
    return 0
  fi
  if [[ -n "${ASSET_SEAL_SALT:-}" ]] || [[ -n "${AD_EVENT_PROCESSOR_ASSET_SEAL_SALT:-}" ]]; then
    return 0
  fi
  if [[ "${RELEASE_GARBLE_SKIP_SEED:-}" == "1" ]]; then
    echo "release_garble: ASSET_SEAL_SALT unset (RELEASE_GARBLE_SKIP_SEED=1 local dev only)" >&2
    return 0
  fi
  return 1
}
