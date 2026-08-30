-- Redis config:values mirror into ngx.shared.edge_config for edge RL, route gates, ASN bypass.
-- Runtime: worker 0 timer CONFIG_SYNC_INTERVAL 5 s (init-worker.lua sync_edge_config); all workers read on access phase.
--
-- Consumers:
-- - edge-rl.lua: get(), get_tier_pct(), get_retry_after().
-- - edge-route-gate.lua: get_flag(edge_expose_click|edge_expose_openrtb) with env fallback.
-- - edge-asn.lua: asn_whitelisted() reads asn_cdn:* / asn_mobile:* generation stamps.
--
-- Memory model (generational ASN, no get_keys):
-- - _asn_ver (number): monotonic generation; asn_whitelisted true when asn_*:{asn} == _asn_ver.
-- - asn_cdn:{asn}, asn_mobile:{asn} (number): generation stamp at sync time, not boolean.
-- - Full sync: compute new_ver, stamp all CSV members with new_ver, bump _asn_ver last (no TOCTOU).
-- - Unblock/clear: stale stamp (asn_ver ~= _asn_ver); no per-ASN delete. Empty CSV or missing field
--   bumps _asn_ver without new stamps so prior members go stale.
-- - RL numerics (limit_per_min, window_ms, rl_pct_*, retry_*_sec): retain prior SHM on null/missing
--   (fail-open); set only when redis_value_ok and tonumber succeeds.
-- - edge_expose_* flags: truthy -> 1; explicit false -> 0; null/ngx.null/missing -> dict:delete (env fallback).
--
-- Redis connect/HMGET fail: fail-open on prior dict (no flush); route gates may fall back to env.
--
-- ngx.shared edge_config (types):
-- - _asn_ver (number): active ASN whitelist generation.
-- - limit_per_min (number): base per-campaign RL per minute; default 100 when missing or <= 0.
-- - window_ms (number): RL window milliseconds; default 60000.
-- - rl_pct_suspect (number): tier scale percent; default 50.
-- - rl_pct_ivt (number): default 10.
-- - rl_pct_block (number): default 0 (block tier -> limit 0).
-- - retry_suspect_sec (number): Retry-After hint; default 30.
-- - retry_ivt_sec (number): default 60.
-- - retry_block_sec (number): default 120.
-- - asn_cdn_raw (string): comma CSV from Redis; source for asn_cdn:* stamps.
-- - asn_mobile_raw (string): comma CSV; source for asn_mobile:* stamps.
-- - asn_cdn:{asn} (number): CDN ASN generation stamp.
-- - asn_mobile:{asn} (number): mobile ASN generation stamp.
-- - edge_expose_click (number 0|1): route gate; deleted when Redis field absent; env when nil.
-- - edge_expose_openrtb (number 0|1): route gate; deleted when Redis field absent; env when nil.
-- - _asn_cdn_count (number): stamped CDN ASN keys at last sync (Prometheus gauge source).
-- - _asn_mobile_count (number): stamped mobile ASN keys at last sync.
--
-- State machine:
-- - Cold boot: worker 0 timer at 0 -> sync(); partial dict until first successful HMGET.
-- - Steady: every 5 s HMGET config:values on connect_any_shard; set only fields with valid Redis values.
-- - Redis down: WARN log, return without mutation; readers use prior SHM or coded defaults.
--
-- Constants and limits:
-- - CONFIG_SYNC_INTERVAL 5 s (init-worker.lua).
-- - DEFAULT_LIMIT 100, DEFAULT_WINDOW_MS 60000.
-- - DEFAULT_SUSPECT_PCT 50, DEFAULT_IVT_PCT 10, DEFAULT_BLOCK_PCT 0.
-- - DEFAULT_RETRY_SUSPECT 30, DEFAULT_RETRY_IVT 60, DEFAULT_RETRY_BLOCK 120.
-- - Redis keepalive 10000 ms, pool 8 after HMGET.
-- - SHM zone edge_config 4m (nginx.conf): asn_* stamps + raw CSV; LRU eviction if undersized.
--   sync() tracks _asn_cdn_count/_asn_mobile_count and logs WARN on dict:set fail or free_space < 8 KiB.
--
-- Forbidden: per-request Redis HMGET from access phase; sync timer only; dict:get_keys for ASN clear.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-config.lua
-- bash scripts/test/edge/lua_tests.sh unit
local blacklist_sync = require "edge-blacklist-sync"
local edge_circuit = require "edge-circuit"

local _M = {}

local dict = ngx.shared.edge_config

local DEFAULT_LIMIT = 100
local DEFAULT_WINDOW_MS = 60000
local DEFAULT_SUSPECT_PCT = 50
local DEFAULT_IVT_PCT = 10
local DEFAULT_BLOCK_PCT = 0
local DEFAULT_RETRY_SUSPECT = 30
local DEFAULT_RETRY_IVT = 60
local DEFAULT_RETRY_BLOCK = 120
local SHM_FREE_WARN_BYTES = 8192

local function redis_value_ok(v)
    if v == nil then
        return false
    end
    if v == ngx.null then
        return false
    end
    if v == "" then
        return false
    end
    return true
end

local function truthy_flag(v)
    if v == true or v == 1 then
        return true
    end
    if type(v) == "number" then
        return v ~= 0
    end
    if type(v) == "string" then
        local s = string.lower(v)
        return s == "1" or s == "true" or s == "yes"
    end
    return false
end

function _M.get()
    local limit = dict:get "limit_per_min"
    local window_ms = dict:get "window_ms"
    if not limit or limit <= 0 then
        limit = DEFAULT_LIMIT
    end
    if not window_ms or window_ms <= 0 then
        window_ms = DEFAULT_WINDOW_MS
    end
    return limit, window_ms
end

function _M.get_tier_pct(tier)
    if tier == "suspect" then
        return dict:get "rl_pct_suspect" or DEFAULT_SUSPECT_PCT
    end
    if tier == "ivt" then
        return dict:get "rl_pct_ivt" or DEFAULT_IVT_PCT
    end
    if tier == "block" then
        return dict:get "rl_pct_block" or DEFAULT_BLOCK_PCT
    end
    return 100
end

function _M.get_retry_after(tier)
    if tier == "block" then
        return dict:get "retry_block_sec" or DEFAULT_RETRY_BLOCK
    end
    if tier == "ivt" then
        return dict:get "retry_ivt_sec" or DEFAULT_RETRY_IVT
    end
    if tier == "suspect" then
        return dict:get "retry_suspect_sec" or DEFAULT_RETRY_SUSPECT
    end
    return dict:get "retry_suspect_sec" or DEFAULT_RETRY_SUSPECT
end

function _M.get_flag(name)
    return dict:get(name)
end

local function stamp_asn_list(raw, prefix, ver)
    if not raw or raw == "" then
        return 0, 0
    end
    local stamped = 0
    local failed = 0
    for asn in string.gmatch(raw, "([^,]+)") do
        asn = string.match(asn, "^%s*(.-)%s*$")
        if asn ~= "" then
            local ok, err = dict:set(prefix .. asn, ver)
            if ok then
                stamped = stamped + 1
            else
                failed = failed + 1
                ngx.log(ngx.WARN, "edge_config: dict:set failed key=", prefix, asn, " err=", err or "no memory")
            end
        end
    end
    return stamped, failed
end

function _M.asn_whitelisted(asn)
    if not asn or asn == "" then
        return false
    end
    asn = string.match(asn, "^%s*(.-)%s*$")
    local ver = dict:get("_asn_ver")
    if not ver then
        return false
    end
    if dict:get("asn_cdn:" .. asn) == ver then
        return true
    end
    if dict:get("asn_mobile:" .. asn) == ver then
        return true
    end
    return false
end

local function sync_expose_flag(field, val)
    if not redis_value_ok(val) then
        dict:delete(field)
        return
    end
    if truthy_flag(val) then
        dict:set(field, 1)
    else
        dict:set(field, 0)
    end
end

function _M.sync()
    local red, err = blacklist_sync.connect_any_shard()
    if not red then
        edge_circuit.record_err()
        ngx.log(ngx.WARN, "edge_config: redis connect failed: ", err)
        return
    end

    local vals, cmd_err = red:hmget(
        "config:values",
        "rate_limit_per_min",
        "rate_limit_window_ms",
        "fraud_rl_suspect_pct",
        "fraud_rl_ivt_pct",
        "fraud_rl_block_pct",
        "fraud_rl_retry_suspect_sec",
        "fraud_rl_retry_ivt_sec",
        "fraud_rl_retry_block_sec",
        "asn_cdn_whitelist",
        "asn_mobile_whitelist",
        "edge_expose_click",
        "edge_expose_openrtb"
    )
    red:set_keepalive(10000, 8)
    if not vals or type(vals) ~= "table" then
        edge_circuit.record_err()
        ngx.log(ngx.WARN, "edge_config: hmget failed: ", cmd_err or "empty")
        return
    end

    if redis_value_ok(vals[1]) then
        local limit = tonumber(vals[1])
        if limit and limit > 0 then
            dict:set("limit_per_min", limit)
        end
    end
    if redis_value_ok(vals[2]) then
        local window_ms = tonumber(vals[2])
        if window_ms and window_ms > 0 then
            dict:set("window_ms", window_ms)
        end
    end

    if redis_value_ok(vals[3]) then
        local suspect_pct = tonumber(vals[3])
        if suspect_pct then
            dict:set("rl_pct_suspect", suspect_pct)
        end
    end
    if redis_value_ok(vals[4]) then
        local ivt_pct = tonumber(vals[4])
        if ivt_pct then
            dict:set("rl_pct_ivt", ivt_pct)
        end
    end
    if redis_value_ok(vals[5]) then
        local block_pct = tonumber(vals[5])
        if block_pct then
            dict:set("rl_pct_block", block_pct)
        end
    end

    if redis_value_ok(vals[6]) then
        local retry_suspect = tonumber(vals[6])
        if retry_suspect and retry_suspect > 0 then
            dict:set("retry_suspect_sec", retry_suspect)
        end
    end
    if redis_value_ok(vals[7]) then
        local retry_ivt = tonumber(vals[7])
        if retry_ivt and retry_ivt > 0 then
            dict:set("retry_ivt_sec", retry_ivt)
        end
    end
    if redis_value_ok(vals[8]) then
        local retry_block = tonumber(vals[8])
        if retry_block and retry_block > 0 then
            dict:set("retry_block_sec", retry_block)
        end
    end

    local new_asn_ver = (dict:get("_asn_ver") or 0) + 1
    local cdn_stamped, cdn_failed = 0, 0
    local mobile_stamped, mobile_failed = 0, 0
    if redis_value_ok(vals[9]) then
        dict:set("asn_cdn_raw", vals[9])
        cdn_stamped, cdn_failed = stamp_asn_list(vals[9], "asn_cdn:", new_asn_ver)
    elseif vals[9] == "" then
        dict:delete("asn_cdn_raw")
    end
    dict:set("_asn_cdn_count", cdn_stamped)
    if redis_value_ok(vals[10]) then
        dict:set("asn_mobile_raw", vals[10])
        mobile_stamped, mobile_failed = stamp_asn_list(vals[10], "asn_mobile:", new_asn_ver)
    elseif vals[10] == "" then
        dict:delete("asn_mobile_raw")
    end
    dict:set("_asn_mobile_count", mobile_stamped)
    dict:set("_asn_ver", new_asn_ver)

    sync_expose_flag("edge_expose_click", vals[11])
    sync_expose_flag("edge_expose_openrtb", vals[12])

    local free_bytes = dict:free_space()
    if cdn_failed > 0 or mobile_failed > 0 or (free_bytes and free_bytes < SHM_FREE_WARN_BYTES) then
        ngx.log(
            ngx.WARN,
            "edge_config: shm pressure cdn_stamped=",
            cdn_stamped,
            " cdn_failed=",
            cdn_failed,
            " mobile_stamped=",
            mobile_stamped,
            " mobile_failed=",
            mobile_failed,
            " free_bytes=",
            free_bytes or "?"
        )
    end
end

-- Test-only exports (edge_config_test.lua).
_M.redis_value_ok = redis_value_ok
_M.truthy_flag = truthy_flag

return _M
