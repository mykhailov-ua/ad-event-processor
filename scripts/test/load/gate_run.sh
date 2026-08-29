#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

BASELINE_REF="${BASELINE_REF:-main}"
BASELINE_WORKTREE="${BASELINE_WORKTREE:-$ROOT/.cache/perf-baseline-worktree}"
OUTDIR="${OUTDIR:-$CI_ARTIFACT_DIR/perf-gate}"
mkdir -p "$OUTDIR"
STRICT="${PERF_GATE_STRICT:-true}"

if [[ "$BASELINE_WORKTREE" != /* ]]; then
  BASELINE_WORKTREE="$ROOT/$BASELINE_WORKTREE"
fi
if [[ "$STRICT" == "true" ]]; then
  BASELINE_WORKTREE="$(safe_worktree_dir "$BASELINE_WORKTREE")"
fi

PR_BENCH="$OUTDIR/pr_bench.txt"
BASELINE_BENCH="$OUTDIR/baseline_bench.txt"
GATE_REPORT="$OUTDIR/gate_report.txt"

"$SCRIPTS/perf/install_benchstat.sh"

echo "perf-gate-run: generating sqlc on current tree..."
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 generate
"$SCRIPTS/test/load/gate_bench.sh" > "$PR_BENCH"

if [[ "$STRICT" != "true" ]]; then
  echo "perf-gate-run: smoke mode - alloc gate NOT run (set PERF_GATE_STRICT=true for strict benchstat + alloc regression)"
  tail -5 "$PR_BENCH"
  exit 0
fi

echo "perf-gate-run: strict mode - baseline ref=$BASELINE_REF worktree=$BASELINE_WORKTREE"
git worktree prune || true
safe_rm_rf "$BASELINE_WORKTREE"
git worktree add --detach "$BASELINE_WORKTREE" "$BASELINE_REF"

(
  cd "$BASELINE_WORKTREE"
  go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 generate
  "$SCRIPTS/test/load/gate_bench.sh" > "$BASELINE_BENCH"
)

git worktree remove --force "$BASELINE_WORKTREE" 2> /dev/null || safe_rm_rf "$BASELINE_WORKTREE"

go run ./cmd/perf-gate "$BASELINE_BENCH" "$PR_BENCH" > "$GATE_REPORT"
cat "$GATE_REPORT"
