#!/usr/bin/env bash

# Role: Library: Dev compose bind mount helpers.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/dev_bind_mounts.sh
aed_compose_dev_overlay() {
  if installer_use_release_images; then
    return 1
  fi
  local profile
  profile="$(installer_env_dual AD_EVENT_PROCESSOR_PROFILE AD_EVENT_PROCESSOR_PROFILE)"
  [[ "$profile" != "production" ]]
}

dev_monorepo_root() {
  dirname "${ROOT:-.}"
}

dev_remove_path() {
  local target="$1"
  local parent base
  [[ -e "$target" ]] || return 0
  parent="$(dirname "$target")"
  base="$(basename "$target")"
  if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
    docker run --rm -v "$parent:/work:rw" alpine rm -rf "/work/$base" 2> /dev/null || true
  fi
  if [[ -e "$target" ]]; then
    rm -rf "$target" 2> /dev/null || true
  fi
  if [[ -e "$target" ]]; then
    echo "dev_bind_mounts: WARN could not remove $target (permission denied?)" >&2
    echo "dev_bind_mounts: try: docker run --rm -v \"$parent:/work:rw\" alpine rm -rf /work/$base" >&2
    return 1
  fi
}

dev_prune_parent_bind_artifacts() {
  local parent name path
  parent="$(dev_monorepo_root)"
  for name in deploy license.jwt var; do
    path="$parent/$name"
    [[ -e "$path" ]] || continue
    case "$path" in
      "$ROOT" | "$ROOT"/*) continue ;;
    esac
    echo "dev_bind_mounts: remove stale parent bind-mount artifact: $path" >&2
    dev_remove_path "$path" || true
  done
}

dev_fix_repo_bind_artifacts() {
  local lic="$ROOT/var/license.jwt"
  if [[ -e "$lic" && -d "$lic" ]]; then
    echo "dev_bind_mounts: $lic is a directory (bind-mount artifact)" >&2
    dev_remove_path "$lic" || true
  fi
  if [[ -e "$ROOT/license.jwt" ]]; then
    echo "dev_bind_mounts: remove repo-root license.jwt (use var/license.jwt)" >&2
    dev_remove_path "$ROOT/license.jwt" || true
  fi
}

dev_prepare_compose_mounts() {
  dev_prune_parent_bind_artifacts
  if aed_compose_dev_overlay; then
    dev_fix_repo_bind_artifacts
  fi
}

dev_finalize_compose_mounts() {
  dev_prune_parent_bind_artifacts
  if aed_compose_dev_overlay; then
    dev_fix_repo_bind_artifacts
  fi
}
