#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
LUA_MOUNT="/etc/nginx/lua"

run_tests() {
	docker run --rm -v "$ROOT/deploy/nginx/lua:${LUA_MOUNT}:ro" openresty/openresty:alpine \
		/usr/local/openresty/luajit/bin/luajit "${LUA_MOUNT}/edge_tls_fingerprint_test.lua" "${LUA_MOUNT}"
}

run_tests
echo "edge_tls_fingerprint_test: PASS"
