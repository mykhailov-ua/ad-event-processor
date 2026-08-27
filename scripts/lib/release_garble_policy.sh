#!/usr/bin/env bash

# Return 0 when garble release build may proceed; 1 when GARBLE_SEED is required but missing.
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

# Return 0 when release asset seal salt is present; 1 when required but missing.
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
