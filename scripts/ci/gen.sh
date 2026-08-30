#!/usr/bin/env bash
set -euo pipefail

# Role: CI codegen entry: sqlc, traffic templates, optional proto/templ/bpf via flags.
# Execution context: CI integration/full_test and operator after schema/proto changes.
# Invariants/contracts enforced: safe_validate_codegen_configs before writes; bpf skipped when clang missing.
# Verify: bash scripts/ci/gen.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

safe_validate_codegen_configs

RUN_PROTO=0
RUN_TEMPL=0
RUN_BPF=0

for arg in "$@"; do
  case "$arg" in
    --proto) RUN_PROTO=1 ;;
    --templ) RUN_TEMPL=1 ;;
    --bpf) RUN_BPF=1 ;;
    --all)
      RUN_PROTO=1
      RUN_TEMPL=1
      RUN_BPF=1
      ;;
  esac
done

echo "gen: sqlc..."
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 generate

echo "gen: traffic source templates..."
go run ./cmd/codegen-traffic-templates

# templ generate optional; skipped when binary missing
if [[ "$RUN_TEMPL" -eq 1 ]]; then
  if command -v templ > /dev/null 2>&1; then
    echo "gen: templ..."
    templ generate
  else
    echo "gen: skip templ (binary not installed)" >&2
  fi
fi

# buf generate only when --proto or --all
if [[ "$RUN_PROTO" -eq 1 ]]; then
  echo "gen: buf (messages)..."
  safe_rm_rf "$ROOT/api/gen"
  mkdir -p "$ROOT/api/gen"
  (cd "$ROOT/api" && go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf.gen.nogrpc.yaml .)
  echo "gen: buf (vtproto: events, vast)..."
  (cd "$ROOT/api" && go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf.gen.vtproto.yaml --path events.proto --path vast.proto)
  safe_sync_proto_gen
  safe_prune_service_vtproto
  echo "gen: patch vtproto hot path..."
  (cd "$ROOT" && go run ./cmd/patch-vtproto-hotpath)
fi

# bpf2go only when --bpf or --all; clang optional on laptop
if [[ "$RUN_BPF" -eq 1 ]]; then
  if command -v clang > /dev/null 2>&1; then
    echo "gen: bpf2go..."
    (cd internal/edge && go generate ./...)
  else
    echo "gen: skip bpf (clang not installed; use committed edge_*.o or Docker bpf2go)" >&2
  fi
fi
