-- Circuit breaker sampling for ngx.shared.circuit_breaker (10 s buckets, key TTL 30 s on incr).
-- Runtime: nginx worker Lua VM; writers on timer sync failures, blacklist stale exits, log_by_lua upstream 5xx.
--
-- Keys (bucket = floor(ngx.time() / 10)):
-- - {bucket}:total incremented once per edge request (access-check.lua record_total).
-- - {bucket}:errs incremented on infra failure paths listed below.
--
-- Writers:
-- - record_total: access-check perimeter_gate (every request).
-- - record_err: edge-blacklist-sync connect_any_shard exhaustion, sync connect/smembers fail;
--   edge-config sync connect/hmget fail; access-check perimeter_blacklist stale/missing sync;
--   edge-circuit-log upstream 5xx or empty upstream_status when upstream_addr set.
--
-- open(): errs/(total_curr+total_prev) > 0.95 after SAMPLE_WINDOW 100 combined samples; else fail-open.
--
-- Forbidden: record_err on edge-generated 503 without upstream (circuit/blacklist) via log_by_lua.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-circuit.lua
-- luajit deploy/nginx/lua/tests/circuit_breaker_test.lua deploy/nginx/lua
-- bash scripts/test/edge/lua_tests.sh
local _M = {}

local circuit_dict = ngx.shared.circuit_breaker

local BUCKET_SEC = 10
local KEY_TTL = 30
local FAIL_THRESHOLD = 0.95
local SAMPLE_WINDOW = 100

function _M.buckets()
    local curr = math.floor(ngx.time() / BUCKET_SEC)
    return curr, curr - 1
end

function _M.record_total()
    local bucket_curr = _M.buckets()
    circuit_dict:incr(bucket_curr .. ":total", 1, 0, KEY_TTL)
end

function _M.record_err()
    local bucket_curr = _M.buckets()
    circuit_dict:incr(bucket_curr .. ":errs", 1, 0, KEY_TTL)
end

function _M.open(bucket_curr, bucket_prev)
    local total_curr = circuit_dict:get(bucket_curr .. ":total") or 0
    local total_prev = circuit_dict:get(bucket_prev .. ":total") or 0
    local total_reqs = total_curr + total_prev
    if total_reqs <= SAMPLE_WINDOW then
        return false
    end
    local errs_curr = circuit_dict:get(bucket_curr .. ":errs") or 0
    local errs_prev = circuit_dict:get(bucket_prev .. ":errs") or 0
    local redis_errs = errs_curr + errs_prev
    return (redis_errs / total_reqs) > FAIL_THRESHOLD
end

function _M.log_upstream_err()
    local upstream_addr = ngx.var.upstream_addr
    if not upstream_addr or upstream_addr == "" then
        return
    end
    local status = ngx.var.upstream_status
    if not status or status == "" then
        _M.record_err()
        return
    end
    for code in string.gmatch(status, "%d+") do
        if tonumber(code) >= 500 then
            _M.record_err()
            return
        end
    end
end

return _M
