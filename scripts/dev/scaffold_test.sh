#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scaffold_test.sh <package-path> [--scenario <name>] [--force]

Generate integration test boilerplate for a flat internal service package.

Examples:
  bash scripts/dev/scaffold_test.sh internal/notify
  bash scripts/dev/scaffold_test.sh notify --scenario enqueue
  task test-gen -- internal/notify

The generator reuses existing package test helpers when present and falls back to
internal/testutil testcontainers setup. Generated tests include:
  - explicit integration skip reason (merge-pr-fast gate)
  - held-out negative case (fails if validation is removed)
  - real infra wiring (no DB/Redis mocks)
EOF
}

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

FORCE=0
SCENARIO="scaffold"
PKG_INPUT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h | --help)
      usage
      exit 0
      ;;
    --force)
      FORCE=1
      shift
      ;;
    --scenario)
      SCENARIO="${2:-}"
      if [[ -z "$SCENARIO" ]]; then
        echo "scaffold_test: --scenario requires a value" >&2
        exit 1
      fi
      shift 2
      ;;
    --)
      shift
      PKG_INPUT="${1:-}"
      shift
      ;;
    -*)
      echo "scaffold_test: unknown flag $1" >&2
      usage
      exit 1
      ;;
    *)
      if [[ -z "$PKG_INPUT" ]]; then
        PKG_INPUT="$1"
      else
        echo "scaffold_test: unexpected argument $1" >&2
        usage
        exit 1
      fi
      shift
      ;;
  esac
done

if [[ -z "$PKG_INPUT" ]]; then
  usage
  exit 1
fi

PKG_PATH="$PKG_INPUT"
PKG_PATH="${PKG_PATH#./}"
PKG_PATH="${PKG_PATH%/}"

case "$PKG_PATH" in
  internal/*) ;;
  *)
    PKG_PATH="internal/$PKG_PATH"
    ;;
esac

if [[ ! -d "$PKG_PATH" ]]; then
  echo "scaffold_test: package directory not found: $PKG_PATH" >&2
  exit 1
fi

SERVICE_NAME="${PKG_PATH#internal/}"
if [[ "$SERVICE_NAME" == "$PKG_PATH" || "$SERVICE_NAME" == *"/"* ]]; then
  echo "scaffold_test: only flat internal/<service> packages are supported (got $PKG_PATH)" >&2
  exit 1
fi

PKG_GO_FILE="$(find "$PKG_PATH" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' | sort | head -n 1 || true)"
if [[ -z "$PKG_GO_FILE" ]]; then
  echo "scaffold_test: no non-test Go files in $PKG_PATH" >&2
  exit 1
fi

PKG_NAME="$(awk '/^package / {print $2; exit}' "$PKG_GO_FILE")"
if [[ -z "$PKG_NAME" ]]; then
  echo "scaffold_test: could not read package name from $PKG_GO_FILE" >&2
  exit 1
fi

SERVICE_TITLE="$(printf '%s' "$SERVICE_NAME" | sed -E 's/(^|-)([a-z])/\U\2/g' | sed 's/-//g')"
SCENARIO_TITLE="$(printf '%s' "$SCENARIO" | sed -E 's/(^|_|-)([a-z])/\U\2/g' | sed 's/[-_]//g')"

TEST_FILE="$PKG_PATH/${SERVICE_NAME}_integration_test.go"
HELPERS_FILE="$PKG_PATH/integration_helpers_test.go"

if [[ -f "$TEST_FILE" && "$FORCE" -ne 1 ]]; then
  echo "scaffold_test: $TEST_FILE already exists (pass --force to overwrite)" >&2
  exit 1
fi

HAS_MIGRATIONS=0
if [[ -d "$PKG_PATH/migrations" ]] && compgen -G "$PKG_PATH/migrations/*.sql" > /dev/null; then
  HAS_MIGRATIONS=1
fi

USES_REDIS=0
if rg -q 'github.com/redis/go-redis/v9' "$PKG_PATH" --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  USES_REDIS=1
fi

USES_BUDGET_INVARIANT=0
if rg -q 'AssertBudgetInvariant|VerifyBudgetInvariant' "$PKG_PATH" --glob '*.go' 2> /dev/null; then
  USES_BUDGET_INVARIANT=1
fi

HOT_PATH=0
case "$PKG_PATH" in
  internal/ingestion | internal/domain | internal/rtb | cmd/tracker | pkg/broker) HOT_PATH=1 ;;
esac

SETUP_FUNC=""
SETUP_RETURNS_POOL=0
if rg -q 'func setupTestDB\(' "$PKG_PATH" --glob '*_test.go' 2> /dev/null; then
  SETUP_FUNC="setupTestDB"
  SETUP_RETURNS_POOL=1
elif rg -q 'func setupAuthTestInfra\(' "$PKG_PATH" --glob '*_test.go' 2> /dev/null; then
  SETUP_FUNC="setupAuthTestInfra"
elif rg -q 'database\.SetupTestDB\(' "$PKG_PATH" --glob '*_test.go' 2> /dev/null; then
  SETUP_FUNC="database.SetupTestDB"
  SETUP_RETURNS_POOL=1
elif rg -q 'testutil\.SetupAdsPostgres\(' "$PKG_PATH" --glob '*_test.go' 2> /dev/null; then
  SETUP_FUNC="testutil.SetupAdsPostgres"
  SETUP_RETURNS_POOL=1
elif rg -q 'testutil\.SetupPostgres\(' "$PKG_PATH" --glob '*_test.go' 2> /dev/null; then
  SETUP_FUNC="testutil.SetupPostgres"
  SETUP_RETURNS_POOL=1
fi

GENERATE_HELPERS=0
if [[ "$HAS_MIGRATIONS" -eq 1 && -z "$SETUP_FUNC" ]]; then
  GENERATE_HELPERS=1
  SETUP_FUNC="setupIntegrationPostgres"
  SETUP_RETURNS_POOL=1
fi

if [[ "$HAS_MIGRATIONS" -eq 0 && "$USES_REDIS" -eq 0 ]]; then
  echo "scaffold_test: refusing to generate integration test without migrations/ or Redis usage in $PKG_PATH" >&2
  echo "scaffold_test: add schema migrations or wire Redis before generating integration tests" >&2
  exit 1
fi

SVC_FACTORY=""
if rg -q 'func newTestService\(' "$PKG_PATH" --glob '*_test.go' 2> /dev/null; then
  SVC_FACTORY="newTestService(pool)"
elif rg -q 'func NewService\(pool \*pgxpool\.Pool\) \*Service' "$PKG_PATH" --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  SVC_FACTORY="NewService(pool)"
elif rg -q 'func NewService\(' "$PKG_PATH" --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  echo "scaffold_test: found NewService with extra dependencies; add newTestService(pool) helper in test_helpers_test.go" >&2
  echo "scaffold_test: example: func newTestService(pool *pgxpool.Pool) *Service { return NewService(pool, newTestConfig(), newTestBreakers()) }" >&2
  exit 1
else
  echo "scaffold_test: no Service constructor found in $PKG_PATH" >&2
  exit 1
fi

HELD_OUT_METHOD=""
HELD_OUT_ARG=""
HELD_OUT_ERR=""
SMOKE_METHOD=""
SMOKE_ARG=""
SMOKE_ERR=""
if rg -q 'func \(.* \*Service\) GetNotification\(ctx context\.Context, notificationID string\)' "$PKG_PATH" --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  HELD_OUT_METHOD="GetNotification"
  HELD_OUT_ARG='""'
  HELD_OUT_ERR="ErrInvalidNotificationID"
  SMOKE_METHOD="GetNotification"
  SMOKE_ARG='"00000000-0000-0000-0000-000000000000"'
  SMOKE_ERR="ErrNotificationNotFound"
elif rg -q 'func \(.* \*Service\) Start\(ctx context\.Context\) error' "$PKG_PATH" --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  HELD_OUT_METHOD="Start"
  SMOKE_METHOD="Start"
fi

IMPORTS=(
  '"context"'
  '"testing"'
)
if [[ "$SETUP_FUNC" == setupIntegrationPostgres || "$USES_REDIS" -eq 1 ]]; then
  IMPORTS+=('"github.com/bidshard/ad-event-processor/internal/testutil"')
fi
IMPORTS+=('"github.com/stretchr/testify/require"')

SCHEMA_NAME="$(echo "$PKG_NAME" | tr '-' '_')"

if [[ "$GENERATE_HELPERS" -eq 1 && ( ! -f "$HELPERS_FILE" || "$FORCE" -eq 1 ) ]]; then
  cat > "$HELPERS_FILE" <<EOF
package $PKG_NAME

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupIntegrationPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.ServiceMigrationsDir("$SERVICE_NAME")}
	return testutil.SetupPostgres(t, cfg)
}
EOF
  echo "scaffold_test: wrote $HELPERS_FILE"
fi

{
  if [[ "$HOT_PATH" -eq 1 ]]; then
    cat <<'EOF'
// Hot-path package: integration tests prove wiring only; cite make test-alloc-gate for SLA evidence.
EOF
  fi

  cat <<EOF
package $PKG_NAME

import (
EOF

  for imp in "${IMPORTS[@]}"; do
    printf '\t%s\n' "$imp"
  done

  cat <<EOF
)

const integrationSkipReason = "integration: run make test-integration (Docker testcontainers)"

func Test${SERVICE_TITLE}_${SCENARIO_TITLE}_integration(t *testing.T) {
	if testing.Short() {
		t.Skip(integrationSkipReason)
	}
EOF

  if [[ "$SETUP_FUNC" == setupAuthTestInfra ]]; then
    cat <<EOF

	infra, cleanup := setupAuthTestInfra(t)
	defer cleanup()

	svc := infra.newService(t)
	ctx := context.Background()
	_ = svc
	_ = ctx
	require.NotNil(t, infra.Pool, "integration scaffold must exercise real Postgres")
EOF
  elif [[ "$SETUP_RETURNS_POOL" -eq 1 ]]; then
    cat <<EOF

	pool, cleanup := ${SETUP_FUNC}(t)
	defer cleanup()

	svc := ${SVC_FACTORY}
	ctx := context.Background()
EOF
    if [[ -n "$SMOKE_METHOD" && "$SMOKE_METHOD" == GetNotification ]]; then
      cat <<EOF

	_, err := svc.${SMOKE_METHOD}(ctx, ${SMOKE_ARG})
	require.ErrorIs(t, err, ${SMOKE_ERR}, "integration scaffold must hit Postgres via service API")
EOF
    elif [[ -n "$SMOKE_METHOD" && "$SMOKE_METHOD" == Start ]]; then
      cat <<EOF

	require.NoError(t, svc.Start(ctx))
	require.NoError(t, svc.Stop(ctx))
EOF
    else
      cat <<EOF

	_ = svc
	_ = ctx
	require.NotNil(t, pool, "integration scaffold must exercise real Postgres")
EOF
    fi
  fi

  if [[ "$USES_REDIS" -eq 1 ]]; then
    if [[ "$SETUP_RETURNS_POOL" -ne 1 && "$SETUP_FUNC" != setupAuthTestInfra ]]; then
      cat <<EOF

	rdb, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()
	require.NotNil(t, rdb, "integration scaffold must exercise real Redis")
EOF
    elif [[ "$SETUP_RETURNS_POOL" -eq 1 ]]; then
      cat <<EOF

	rdb, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()
	require.NotNil(t, rdb, "integration scaffold must exercise real Redis")
EOF
    fi
  fi

  if [[ "$USES_BUDGET_INVARIANT" -eq 1 ]]; then
    cat <<'EOF'

	// Replace campaignID with a seeded campaign before enabling budget invariant checks.
	// domain.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)
EOF
  fi

  cat <<EOF
}

func Test${SERVICE_TITLE}_${SCENARIO_TITLE}HeldOut_integration(t *testing.T) {
	if testing.Short() {
		t.Skip(integrationSkipReason)
	}
EOF

  if [[ -n "$HELD_OUT_METHOD" && -n "$HELD_OUT_ERR" ]]; then
    cat <<EOF

	pool, cleanup := ${SETUP_FUNC}(t)
	defer cleanup()

	svc := ${SVC_FACTORY}
	ctx := context.Background()

	_, err := svc.${HELD_OUT_METHOD}(ctx, ${HELD_OUT_ARG})
	require.ErrorIs(t, err, ${HELD_OUT_ERR}, "held-out case must fail when input validation is removed")
EOF
  elif [[ "$HELD_OUT_METHOD" == Start ]]; then
    cat <<EOF

	pool, cleanup := ${SETUP_FUNC}(t)
	defer cleanup()

	ctx := context.Background()
	var schemaExists bool
	err := pool.QueryRow(ctx, \`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.schemata WHERE schema_name = \$1
		)\`, "${SCHEMA_NAME}").Scan(&schemaExists)
	require.NoError(t, err)
	require.True(t, schemaExists, "held-out case: service migrations must create schema ${SCHEMA_NAME}")
EOF
  else
    cat <<EOF

	t.Fatal("scaffold_test: add a held-out negative assertion for this package")
EOF
  fi

  cat <<'EOF'
}
EOF
} > "$TEST_FILE"

echo "scaffold_test: wrote $TEST_FILE"
echo "scaffold_test: next steps:"
echo "  1. Replace scaffold placeholders with domain-specific setup and assertions."
echo "  2. go test $PKG_PATH -run '${SERVICE_TITLE}_${SCENARIO_TITLE}' -count=1"
echo "  3. bash scripts/ci/integration_test_slop_gate.sh"
