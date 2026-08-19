#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "clean-gitignored: git clean -fdX..."
git clean -fdX 2>&1 || true

# Docker bind mounts and load-test runs often leave root-owned / nobody-owned trees.
if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
  echo "clean-gitignored: removing permission-blocked ignored paths via docker..."
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

remaining="$(find . -path ./.git -prune -o -type f -print 2>/dev/null | git check-ignore --stdin 2>/dev/null | wc -l || true)"
ignored_untracked="$(git ls-files --others -i --exclude-standard 2>/dev/null | wc -l || true)"
remaining="${remaining//[[:space:]]/}"
ignored_untracked="${ignored_untracked//[[:space:]]/}"

if [[ "$remaining" -ne 0 || "$ignored_untracked" -ne 0 ]]; then
  echo "clean-gitignored: FAILED — ignored files remain on disk" >&2
  find . -path ./.git -prune -o -type f -print 2>/dev/null | git check-ignore --stdin 2>/dev/null | head -20 >&2 || true
  git ls-files --others -i --exclude-standard 2>/dev/null | head -20 >&2 || true
  echo "hint: sudo rm -rf <path> for root-owned artifacts, then re-run: bash scripts/clean/gitignored.sh" >&2
  exit 1
fi

echo "clean-gitignored: OK (sources only)"
