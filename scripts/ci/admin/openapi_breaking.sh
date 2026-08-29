#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

OASDIFF_VERSION="${OASDIFF_VERSION:-v1.29.1}"
OASDIFF_MODULE="github.com/oasdiff/oasdiff@${OASDIFF_VERSION}"
FIXTURE_DIR="$ROOT/internal/openapi/testdata/breaking"
BASE_FIXTURE="$FIXTURE_DIR/base.yaml"
HEAD_FIXTURE="$FIXTURE_DIR/head_removed_field.yaml"
CURRENT_BUNDLE="$ROOT/api/openapi/openapi.bundle.yaml"
ERR_IGNORE="$ROOT/api/openapi/breaking_err_ignore.txt"

run_oasdiff() {
  go run "$OASDIFF_MODULE" "$@"
}

oasdiff_breaking() {
  local base="$1"
  local revision="$2"
  local -a extra=()
  if [[ -s "$ERR_IGNORE" ]] && grep -qvE '^[[:space:]]*(#|$)' "$ERR_IGNORE"; then
    extra+=(--err-ignore "$ERR_IGNORE")
  fi
  run_oasdiff breaking --fail-on ERR "${extra[@]}" "$base" "$revision"
}

echo "openapi_breaking_gate: fixture proves removed-field detection..."
if oasdiff_breaking "$BASE_FIXTURE" "$HEAD_FIXTURE"; then
  echo "openapi_breaking_gate: expected breaking changes between fixture base and head" >&2
  exit 1
fi
if ! oasdiff_breaking "$BASE_FIXTURE" "$BASE_FIXTURE"; then
  echo "openapi_breaking_gate: identical fixture specs must pass" >&2
  exit 1
fi

if [[ "${OPENAPI_BREAKING_SKIP:-}" == "1" ]]; then
  echo "openapi_breaking_gate: skip PR diff (OPENAPI_BREAKING_SKIP=1)"
  exit 0
fi

if [[ ! -f "$CURRENT_BUNDLE" ]]; then
  echo "openapi_breaking_gate: missing $CURRENT_BUNDLE; run make openapi-export first" >&2
  exit 1
fi

DIFF_BASE="${OPENAPI_DIFF_BASE:-}"
if [[ -z "$DIFF_BASE" ]]; then
  for candidate in origin/main origin/master main master; do
    if git rev-parse --verify "$candidate" > /dev/null 2>&1; then
      DIFF_BASE="$(git merge-base HEAD "$candidate" 2> /dev/null || true)"
      if [[ -n "$DIFF_BASE" ]]; then
        break
      fi
    fi
  done
fi

if [[ -z "$DIFF_BASE" ]]; then
  echo "openapi_breaking_gate: no merge base; fixture OK, PR diff skipped"
  exit 0
fi

if ! git cat-file -e "${DIFF_BASE}:api/openapi/openapi.yaml" 2> /dev/null; then
  echo "openapi_breaking_gate: base commit has no api/openapi/openapi.yaml; PR diff skipped"
  exit 0
fi

WORKTREE=""
cleanup() {
  if [[ -n "$WORKTREE" && -d "$WORKTREE" ]]; then
    git worktree remove "$WORKTREE" --force > /dev/null 2>&1 || rm -rf "$WORKTREE"
  fi
}
trap cleanup EXIT

WORKTREE="$(mktemp -d)"
if ! git worktree add --detach "$WORKTREE" "$DIFF_BASE" > /dev/null 2>&1; then
  echo "openapi_breaking_gate: could not attach worktree at $DIFF_BASE; PR diff skipped"
  exit 0
fi

echo "openapi_breaking_gate: export base bundle at ${DIFF_BASE:0:12}..."
if ! (cd "$WORKTREE" && go run ./cmd/openapi-export > /dev/null); then
  echo "openapi_breaking_gate: base openapi-export failed; PR diff skipped" >&2
  exit 0
fi

BASE_BUNDLE="$WORKTREE/api/openapi/openapi.bundle.yaml"
if [[ ! -f "$BASE_BUNDLE" ]]; then
  echo "openapi_breaking_gate: base bundle missing after export; PR diff skipped" >&2
  exit 0
fi

echo "openapi_breaking_gate: compare bundled spec against merge base..."
if ! oasdiff_breaking "$BASE_BUNDLE" "$CURRENT_BUNDLE"; then
  echo "openapi_breaking_gate: breaking OpenAPI changes detected (see oasdiff output above)" >&2
  echo "openapi_breaking_gate: bump admin TS types (make openapi-types) and note the change in release notes" >&2
  exit 1
fi

echo "openapi_breaking_gate: OK"
