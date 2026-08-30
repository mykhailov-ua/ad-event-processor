# shellcheck shell=bash

# Role: Repo path bootstrap: ROOT, SCRIPTS, and ci_artifacts for sourced CI scripts.
# Execution context: Sourced by CI/fault/lib scripts; not executed directly.
# Invariants/contracts enforced: Exports canonical ROOT/SCRIPTS; pulls safe_paths and ci_artifacts.
# Verify: source scripts/lib/paths.sh (via any scripts/ci/*.sh entry)
_AD_EVENT_PROCESSOR_SCRIPTS="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AD_EVENT_PROCESSOR_SCRIPTS_ROOT="$_AD_EVENT_PROCESSOR_SCRIPTS"
AD_EVENT_PROCESSOR_REPO_ROOT="$(cd "$AD_EVENT_PROCESSOR_SCRIPTS_ROOT/.." && pwd)"
ROOT="$AD_EVENT_PROCESSOR_REPO_ROOT"
SCRIPTS="$AD_EVENT_PROCESSOR_SCRIPTS_ROOT"
export ROOT SCRIPTS AD_EVENT_PROCESSOR_SCRIPTS_ROOT AD_EVENT_PROCESSOR_REPO_ROOT

source "$AD_EVENT_PROCESSOR_SCRIPTS_ROOT/lib/safe_paths.sh"
source "$AD_EVENT_PROCESSOR_SCRIPTS_ROOT/lib/ci_artifacts.sh"
