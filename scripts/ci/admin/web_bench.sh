#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

node --expose-gc --experimental-strip-types --import "$ROOT/web/scripts/register_ts.mjs" "$ROOT/web/scripts/bench/run.mjs"
node --expose-gc --experimental-strip-types --import "$ROOT/web/scripts/register_ts.mjs" "$ROOT/web/scripts/bench/hot_path.mjs"
