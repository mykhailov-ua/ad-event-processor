#!/usr/bin/env bash
set -euo pipefail

# Role: Naming gate: Staged files not gitignored incorrectly.
# Execution context: CI via pr_fast or full_test.
# Invariants/contracts enforced: Legacy tokens and layout violations fail closed.
# Verify: bash scripts/ci/naming/staged_not_ignored.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0
while IFS= read -r -d '' path; do
  [[ -n "$path" ]] || continue
  case "$path" in
    .env | .env.secrets)
      echo "check_staged_not_ignored: $path must not be committed - unstage (git reset HEAD -- \"$path\")" >&2
      fail=1
      continue
      ;;
  esac
  if git check-ignore -q "$path"; then
    echo "check_staged_not_ignored: $path is gitignored - unstage (git reset HEAD -- \"$path\")" >&2
    fail=1
  fi
done < <(git diff --cached --name-only -z --diff-filter=ACMR 2> /dev/null || true)

# Aggregate fail flag: exit 1 after all checks
if [[ "$fail" -ne 0 ]]; then
  echo "check_staged_not_ignored: FAILED - do not commit build/cache artifacts" >&2
  exit 1
fi

echo "check_staged_not_ignored: OK"
