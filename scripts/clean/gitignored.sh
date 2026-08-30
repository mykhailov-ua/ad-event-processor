#!/usr/bin/env bash
set -euo pipefail

# Role: Remove gitignored build artifacts and permission-blocked paths (bin, var, .cache) via git clean and docker alpine.
# Execution context: Operator cleanup only; destructive to ignored generated files.
# Invariants/contracts enforced: dev_prune_parent_bind_artifacts runs first; never touches tracked sources.
# Verify: bash scripts/clean/gitignored.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/dev_bind_mounts.sh"
cd "$ROOT"

dev_prune_parent_bind_artifacts

echo "clean-gitignored: git clean -fdX..."
git clean -fdX 2>&1 || true

if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
  echo "clean-gitignored: removing permission-blocked ignored paths via docker..."
  MONOREPO_ROOT="$(dirname "$ROOT")"
  docker run --rm -v "$MONOREPO_ROOT:/work:rw" alpine sh -c '
    rm -rf /work/deploy /work/license.jwt /work/var 2>/dev/null || true
  ' || true
  docker run --rm -v "$ROOT:/work" alpine sh -c '
    rm -rf \
      /work/.cache \
      /work/var \
      /work/license.jwt \
      /work/bin \
      /work/node_modules \
      /work/web/node_modules \
      /work/web/dist \
      /work/web/e2e/node_modules \
      /work/web/test-results \
      /work/web/e2e/test-results \
      2>/dev/null || true
  ' || true
  git clean -fdX 2>&1 || true
fi

remaining="$(find . -path ./.git -prune -o -type f -print 2> /dev/null | git check-ignore --stdin 2> /dev/null | wc -l || true)"
ignored_untracked="$(git ls-files --others -i --exclude-standard 2> /dev/null | wc -l || true)"
remaining="${remaining//[[:space:]]/}"
ignored_untracked="${ignored_untracked//[[:space:]]/}"

if [[ "$remaining" -ne 0 || "$ignored_untracked" -ne 0 ]]; then
  echo "clean-gitignored: FAILED - ignored files remain on disk" >&2
  find . -path ./.git -prune -o -type f -print 2> /dev/null | git check-ignore --stdin 2> /dev/null | head -20 >&2 || true
  git ls-files --others -i --exclude-standard 2> /dev/null | head -20 >&2 || true
  echo "hint: sudo rm -rf <path> for root-owned artifacts, then re-run: bash scripts/clean/gitignored.sh" >&2
  exit 1
fi

echo "clean-gitignored: OK (sources only)"
