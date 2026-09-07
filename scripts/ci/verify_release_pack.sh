#!/usr/bin/env bash

set -euo pipefail

# Role: Release pack verification.
# Execution context: Release operator.
# Invariants/contracts enforced: Missing release artifacts fail.
# Verify: bash scripts/ci/verify_release_pack.sh
TARBALL="${1:?usage: verify_release_pack.sh <tarball>}"

FORBIDDEN_PATTERNS=(
  'license_private'
  'sku.yaml'
  '/cmd/license-issue'
  'license-issue'
)

LIST="$(tar -tzf "$TARBALL")"
for pat in "${FORBIDDEN_PATTERNS[@]}"; do
  if echo "$LIST" | grep -qi "$pat"; then
    echo "verify_release_pack: forbidden path matching '$pat' in $TARBALL" >&2
    exit 1
  fi
done

if ! echo "$LIST" | grep -q 'deploy/vendor/license_public.key'; then
  echo "verify_release_pack: missing deploy/vendor/license_public.key in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'bin/ad-event-processor-install'; then
  echo "verify_release_pack: missing bin/ad-event-processor-install in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/install/preflight.sh'; then
  echo "verify_release_pack: missing scripts/install/preflight.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'bin/migrate-cold-path'; then
  echo "verify_release_pack: missing bin/migrate-cold-path in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'internal/ingest/migrations/00001_init_schema.sql'; then
  echo "verify_release_pack: missing internal/ingest/migrations in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/ops/bootstrap_pg_schema.sh'; then
  echo "verify_release_pack: missing scripts/ops/bootstrap_pg_schema.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/install/ad-event-processor-install.sh'; then
  echo "verify_release_pack: missing scripts/install/ad-event-processor-install.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/dev/stack/stack.sh'; then
  echo "verify_release_pack: missing scripts/dev/stack/stack.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/lib/install_cli.sh'; then
  echo "verify_release_pack: missing scripts/lib/install_cli.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/install/mode_systemd.sh'; then
  echo "verify_release_pack: missing scripts/install/mode_systemd.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'deploy/systemd/ad-event-processor-control.service'; then
  echo "verify_release_pack: missing deploy/systemd units in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'install.sh'; then
  echo "verify_release_pack: missing install.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'TESTER_QUICKSTART.md'; then
  echo "verify_release_pack: missing TESTER_QUICKSTART.md in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'deploy/compose/scripts/init-run-volume.sh'; then
  echo "verify_release_pack: missing deploy/compose/scripts/init-run-volume.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'bin/broker'; then
  echo "verify_release_pack: missing bin/broker in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/lib/ci_artifacts.sh'; then
  echo "verify_release_pack: missing scripts/lib/ci_artifacts.sh in $TARBALL" >&2
  exit 1
fi

if ! echo "$LIST" | grep -q 'scripts/lib/safe_paths.sh'; then
  echo "verify_release_pack: missing scripts/lib/safe_paths.sh in $TARBALL" >&2
  exit 1
fi

echo "verify_release_pack: OK ($TARBALL)"
