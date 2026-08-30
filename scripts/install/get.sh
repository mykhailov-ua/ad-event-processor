#!/usr/bin/env bash
# Role: Remote installer curl entrypoint; clones or downloads release tarball to INSTALL_DIR.
# Execution context: Operator host with curl/git; not run from repo checkout directly.
# Env knobs: AD_EVENT_PROCESSOR_INSTALL_REPO; AD_EVENT_PROCESSOR_VERSION (latest); AD_EVENT_PROCESSOR_INSTALL_DIR;
#   AD_EVENT_PROCESSOR_INSTALL_FROM_GIT (0).
# Verify: AD_EVENT_PROCESSOR_VERSION=latest bash scripts/install/get.sh --help 2>&1 | head -1
set -euo pipefail

REPO="${AD_EVENT_PROCESSOR_INSTALL_REPO:-ad-event-processor/ad-event-processor}"
VERSION="${AD_EVENT_PROCESSOR_VERSION:-latest}"
INSTALL_DIR="${AD_EVENT_PROCESSOR_INSTALL_DIR:-${HOME}/ad-event-processor}"
USE_GIT="${AD_EVENT_PROCESSOR_INSTALL_FROM_GIT:-0}"
GET_SCRIPT_URL="${AD_EVENT_PROCESSOR_GET_SCRIPT_URL:-}"

log() {
  echo "ad-event-processor-get: $*"
}

need_cmd() {
  if ! command -v "$1" > /dev/null 2>&1; then
    echo "ad-event-processor-get: required command not found: $1" >&2
    exit 1
  fi
}

resolve_latest_tag() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])'
}

install_from_tarball() {
  local tag="$1"
  local tmp
  tmp="$(mktemp -d)"
  for name in ad-event-processor-installer; do
    local url="https://github.com/${REPO}/releases/download/${tag}/${name}.tar.gz"
    log "downloading ${url}"
    if curl -fsSL "$url" | tar -xz -C "$tmp" 2> /dev/null; then
      rm -rf "$INSTALL_DIR"
      mv "$tmp/ad-event-processor" "$INSTALL_DIR"
      rm -rf "$tmp"
      return 0
    fi
  done
  rm -rf "$tmp"
  return 1
}

install_from_git() {
  need_cmd git
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    log "updating existing clone at $INSTALL_DIR"
    cd "$INSTALL_DIR"
    git pull --ff-only
  else
    rm -rf "$INSTALL_DIR"
    log "cloning https://github.com/${REPO}.git"
    git clone "https://github.com/${REPO}.git" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
  fi
  if [[ "$VERSION" != "latest" ]] && [[ "$VERSION" != "main" ]]; then
    git checkout "$VERSION"
  fi
}

main() {
  need_cmd curl
  need_cmd python3

  local tag="$VERSION"
  if [[ "$USE_GIT" == "1" ]]; then
    install_from_git
  elif [[ "$VERSION" == "latest" ]]; then
    tag="$(resolve_latest_tag)"
    if ! install_from_tarball "$tag"; then
      log "release tarball missing; falling back to git clone"
      USE_GIT=1
      install_from_git
    fi
  else
    if ! install_from_tarball "$tag"; then
      log "release tarball missing for ${tag}; falling back to git clone"
      install_from_git
    fi
  fi

  cd "$INSTALL_DIR"
  exec bash scripts/install/ad-event-processor-install.sh --yes "$@"
}

if [[ -n "$GET_SCRIPT_URL" ]]; then
  log "remote script URL set; prefer: curl -fsSL ${GET_SCRIPT_URL} | bash"
fi

main "$@"
