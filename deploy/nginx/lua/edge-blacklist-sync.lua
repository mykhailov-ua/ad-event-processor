-- Redis IP blacklist -> ngx.shared.blacklist_cache via generational versioning (no flush_all, no per-IP delete on unblock).
-- Runtime: worker 0 timers (init-worker.lua); incremental path via edge-quarantine-sub ngx.thread; cosocket Redis.
--
-- Consumer: access-check.lua perimeter_blacklist() reads _bl_ver, _bl_sync_ts, b:{ip}; blocks when ip_ver == ver.
--
-- Generational model:
-- - _bl_ver monotonic generation; b:{ip} stores the generation at stamp time, not a boolean.
-- - Full sync() SMEMBERS blacklist:manual|auto|fraud then bumps _bl_ver and re-stamps every member.
-- - Unblocked IPs are not deleted from SHM; stale b:{ip} with ip_ver ~= ver is harmless (lookup passes).
-- - stamp_ips(ips, bump_version): bump_version true (default) or ver==0 increments _bl_ver; false reuses current ver.
-- - Quarantine pub/sub and drain_pending call stamp_ips(..., false) so incremental adds do not invalidate prior stamps.
--
-- ngx.shared blacklist_cache:
-- - _bl_ver (number): active generation.
-- - _bl_sync_ts (unix s): last successful stamp_ips or sync; access-check stale gate EDGE_BL_STALE_SEC default 30.
-- - _bl_count (number): deduped active IP estimate; full sync() sets exact Redis count;
--   stamp_ips increments only IPs not already at current _bl_ver (no duplicate inflation).
-- - _bl_pending (string): newline-delimited overflow IPs when batch > CHANGELOG_MAX_IPS; drained by init-worker timer.
-- - b:{ip} (number): generation stamp for client IP.
--
-- ngx.shared sentinel_cache:
-- - m:{master_name} -> host:port string, TTL SENTINEL_CACHE_TTL 5 s; invalidated on connect failure.
--
-- State machine:
-- - Cold boot: init-worker timer -> sync() full SMEMBERS; must succeed before access-check serves traffic.
-- - Incremental: fraud:quarantine PUBLISH -> apply_quarantine_message -> stamp_ips(..., false).
-- - Overflow: stamp_ips defers tail to _bl_pending; drain_pending_changelog timer -> stamp_ips(ips, false).
-- - Empty quarantine payload: apply_quarantine_message falls back to full sync().
--
-- Failure branches (prior SHM retained unless noted):
-- - connect_any_shard fail: sync/stamp return false; fail-closed at L7 if _bl_ver or _bl_sync_ts missing/stale.
-- - SMEMBERS fail: sync returns false; prior generation unchanged.
-- - stamp_ips empty input: returns false; no _bl_sync_ts update.
-- - Sentinel resolve fail: WARN, fallback to static REDIS_ADDRS target for that shard.
--
-- Constants: CHANGELOG_MAX_IPS 64 (EDGE_BLACKLIST_CHANGELOG_MAX_IPS); Redis timeout 500 ms; keepalive 10000 ms pool 8.
--
-- Forbidden: flush_all on blacklist_cache; per-IP delete on unblock; claiming L7 cache replaces tracker FilterEngine or XDP for all fraud.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-blacklist-sync.lua
-- bash scripts/test/edge/lua_tests.sh
-- go test ./internal/edge/... -count=1
local redis = require "resty.redis"
local edge_net = require "edge-net"
local edge_circuit = require "edge-circuit"

local _M = {}

local cache = ngx.shared.blacklist_cache
local sentinel_cache = ngx.shared.sentinel_cache

local test_env = nil
local test_connect_shard = nil

local REDIS_HOST = "127.0.0.1" -- fallback TCP when REDIS_ADDRS unset (dev only)
local REDIS_PORT = 6379 -- TCP fallback port; production uses unix sockets in REDIS_ADDRS
local REDIS_PASS = "" -- requirepass; empty in dev compose
local REDIS_ADDRS = "" -- comma unix:/run/.../redis-N.sock; shard try order from edge-net
local REDIS_SENTINEL_ADDRS = "" -- host:26379 list when sentinel failover enabled
local REDIS_MASTER_NAMES = "" -- parallel to sentinel addrs (mymaster per shard)
local SENTINEL_CACHE_TTL = 5 -- seconds; ngx.shared sentinel_cache master resolution cache
local CHANGELOG_MAX_IPS = 64 -- max IPs per fraud:quarantine pub/sub incremental stamp batch

local shards
local sentinel_addrs
local master_names
local sentinel_enabled

local function getenv(name)
    if test_env then
        return test_env(name)
    end
    return os.getenv(name)
end

function _M.set_env_for_test(env)
    test_env = env
    shards = nil
    sentinel_addrs = nil
    master_names = nil
    sentinel_enabled = nil
end

function _M.set_connect_shard_for_test(fn)
    test_connect_shard = fn
end

function _M.reset_test_hooks()
    test_env = nil
    test_connect_shard = nil
    shards = nil
    sentinel_addrs = nil
    master_names = nil
    sentinel_enabled = nil
end

function _M.shard_try_order(shard_count)
    local order = {}
    for i = 1, shard_count do
        order[i] = i
    end
    return order
end

local function load_env()
    REDIS_HOST = getenv "REDIS_HOST" or REDIS_HOST
    REDIS_PORT = getenv "REDIS_PORT" or REDIS_PORT
    REDIS_PASS = getenv "REDIS_PASS" or ""
    REDIS_ADDRS = getenv "REDIS_ADDRS" or ""
    REDIS_SENTINEL_ADDRS = getenv "REDIS_SENTINEL_ADDRS" or ""
    REDIS_MASTER_NAMES = getenv "REDIS_MASTER_NAMES" or ""
    CHANGELOG_MAX_IPS = tonumber(getenv "EDGE_BLACKLIST_CHANGELOG_MAX_IPS" or "") or CHANGELOG_MAX_IPS
end

local function parse_addr_list(raw)
    return edge_net.parse_addr_list(raw)
end

local function sentinel_master_addr(shard_idx, names, sentinels)
    if #names == 0 or #sentinels == 0 then
        return nil, "sentinel not configured"
    end
    if shard_idx < 1 or shard_idx > #names then
        return nil, "shard index out of range"
    end
    local master_name = names[shard_idx]
    local cache_key = "m:" .. master_name
    local cached = sentinel_cache:get(cache_key)
    if cached then
        local host, port = string.match(cached, "([^:]+):(%d+)")
        if host and port then
            return { host = host, port = tonumber(port) }, nil
        end
    end

    local sentinel = sentinels[((shard_idx - 1) % #sentinels) + 1]
    local sred = redis:new()
    sred:set_timeout(200)
    local ok, err = sred:connect(sentinel.host, sentinel.port)
    if not ok then
        return nil, "sentinel connect: " .. (err or "unknown")
    end
    if REDIS_PASS ~= "" then
        local res, auth_err = sred:auth(REDIS_PASS)
        if not res then
            return nil, "sentinel auth: " .. (auth_err or "unknown")
        end
    end

    local res, cmd_err = sred:sentinel("get-master-addr-by-name", master_name)
    sred:set_keepalive(10000, 32)
    if not res or type(res) ~= "table" or #res < 2 then
        return nil, "sentinel get-master-addr-by-name: " .. (cmd_err or "empty response")
    end
    local host = res[1]
    local port = tonumber(res[2])
    if not host or not port then
        return nil, "invalid master addr from sentinel"
    end
    sentinel_cache:set(cache_key, host .. ":" .. port, SENTINEL_CACHE_TTL)
    return { host = host, port = port }, nil
end

local function ensure_redis_topology()
    if shards then
        return
    end
    load_env()
    shards = parse_addr_list(REDIS_ADDRS)
    if #shards == 0 then
        shards = { { host = REDIS_HOST, port = tonumber(REDIS_PORT) } }
    end
    sentinel_addrs = parse_addr_list(REDIS_SENTINEL_ADDRS)
    master_names = {}
    if REDIS_MASTER_NAMES ~= "" then
        for name in string.gmatch(REDIS_MASTER_NAMES, "([^,]+)") do
            name = string.match(name, "^%s*(.-)%s*$")
            if name ~= "" then
                table.insert(master_names, name)
            end
        end
    end
    sentinel_enabled = #sentinel_addrs > 0 and #master_names > 0
end

local function shard_target(shard_idx)
    ensure_redis_topology()
    if shard_idx < 1 or shard_idx > #shards then
        return nil
    end
    local target = shards[shard_idx]
    if sentinel_enabled then
        local resolved, resolve_err = sentinel_master_addr(shard_idx, master_names, sentinel_addrs)
        if resolved then
            target = resolved
        else
            ngx.log(
                ngx.WARN,
                "edge_blacklist_sync: sentinel resolve failed for shard ",
                shard_idx - 1,
                ": ",
                resolve_err
            )
        end
    end
    return target
end

local function connect_shard(shard_idx)
    if test_connect_shard then
        return test_connect_shard(shard_idx)
    end

    local target = shard_target(shard_idx)
    if not target then
        return nil, "invalid shard index"
    end
    local red = redis:new()
    red:set_timeout(500)

    local ok, err = edge_net.redis_connect(red, target)
    if not ok and sentinel_enabled then
        sentinel_cache:delete("m:" .. master_names[shard_idx])
        local resolved, resolve_err = sentinel_master_addr(shard_idx, master_names, sentinel_addrs)
        if resolved then
            target = resolved
            ok, err = edge_net.redis_connect(red, target)
        else
            return nil, "sentinel re-resolve: " .. (resolve_err or "unknown")
        end
    end
    if not ok then
        return nil, err
    end

    if REDIS_PASS ~= "" then
        local res, auth_err = red:auth(REDIS_PASS)
        if not res then
            red:close()
            return nil, auth_err
        end
    end
    return red, nil
end

function _M.connect_any_shard()
    ensure_redis_topology()
    local last_err = "no redis shard configured"
    for _, shard_idx in ipairs(_M.shard_try_order(#shards)) do
        local red, err = connect_shard(shard_idx)
        if red then
            return red, nil, shard_idx
        end
        last_err = err or "connect failed"
        ngx.log(ngx.WARN, "edge_blacklist_sync: shard ", shard_idx - 1, " connect failed: ", last_err)
    end
    edge_circuit.record_err()
    return nil, last_err, nil
end

local function append_pending_ips(ips)
    if not ips or #ips == 0 then
        return
    end
    local raw = cache:get "_bl_pending" or ""
    for _, ip in ipairs(ips) do
        if ip and ip ~= "" then
            raw = raw .. ip .. "\n"
        end
    end
    cache:set("_bl_pending", raw)
end

function _M.stamp_ips(ips, bump_version)
    load_env()
    if not ips or #ips == 0 then
        return false
    end
    if bump_version == nil then
        bump_version = true
    end

    local batch = ips
    local deferred = {}
    if #ips > CHANGELOG_MAX_IPS then
        batch = {}
        for i = 1, CHANGELOG_MAX_IPS do
            batch[i] = ips[i]
        end
        for i = CHANGELOG_MAX_IPS + 1, #ips do
            deferred[#deferred + 1] = ips[i]
        end
    end

    local ver = cache:get "_bl_ver" or 0
    if bump_version or ver == 0 then
        ver = ver + 1
        cache:set("_bl_ver", ver)
    end

    local stamped = 0
    local added = 0
    for _, ip in ipairs(batch) do
        if ip and ip ~= "" then
            local was_active = cache:get("b:" .. ip) == ver
            cache:set("b:" .. ip, ver)
            stamped = stamped + 1
            if not was_active then
                added = added + 1
            end
        end
    end

    if stamped == 0 then
        return false
    end

    if #deferred > 0 then
        append_pending_ips(deferred)
        ngx.log(ngx.WARN, "edge_blacklist_sync: deferred ", #deferred, " changelog IPs (cap=", CHANGELOG_MAX_IPS, ")")
    end

    cache:set("_bl_sync_ts", ngx.time())
    local prev_count = cache:get "_bl_count" or 0
    cache:set("_bl_count", prev_count + added)
    ngx.log(
        ngx.INFO,
        "edge_blacklist_sync: stamped ",
        stamped,
        " IPs (ver=",
        ver,
        ", bump=",
        tostring(bump_version),
        ")"
    )
    return true
end

function _M.drain_pending_changelog()
    local raw = cache:get "_bl_pending"
    if not raw or raw == "" then
        return 0
    end
    local ips = {}
    for ip in string.gmatch(raw, "[^\n]+") do
        ips[#ips + 1] = ip
    end
    if #ips == 0 then
        cache:delete "_bl_pending"
        return 0
    end
    cache:delete "_bl_pending"
    if _M.stamp_ips(ips, false) then
        return math.min(#ips, CHANGELOG_MAX_IPS)
    end
    return 0
end

function _M.apply_quarantine_message(payload)
    if not payload or payload == "" then
        return _M.sync()
    end
    if payload:sub(1, 1) == "{" then
        local cjson = require "cjson.safe"
        local obj = cjson.decode(payload)
        if obj and obj.ips and type(obj.ips) == "table" then
            return _M.stamp_ips(obj.ips, false)
        end
    end
    return _M.stamp_ips({ payload }, false)
end

function _M.sync()
    local red, err, shard_idx = _M.connect_any_shard()
    if not red then
        ngx.log(ngx.WARN, "edge_blacklist_sync: connect failed: ", err)
        return false
    end

    local manual, err1 = red:smembers "blacklist:manual"
    local auto, err2 = red:smembers "blacklist:auto"
    local fraud, err3 = red:smembers "blacklist:fraud"
    red:set_keepalive(10000, 8)

    if not manual or not auto or not fraud then
        edge_circuit.record_err()
        ngx.log(ngx.ERR, "edge_blacklist_sync: smembers failed: ", err1 or err2 or err3)
        return false
    end

    local new_ver = (cache:get "_bl_ver" or 0) + 1
    local count = 0
    local seen = {}

    local function stamp(ip)
        if not ip or ip == "" or seen[ip] then
            return
        end
        seen[ip] = true
        cache:set("b:" .. ip, new_ver)
        count = count + 1
    end

    for _, ip in ipairs(manual) do
        stamp(ip)
    end
    for _, ip in ipairs(auto) do
        stamp(ip)
    end
    for _, ip in ipairs(fraud) do
        stamp(ip)
    end

    cache:set("_bl_ver", new_ver)
    cache:set("_bl_sync_ts", ngx.time())
    cache:set("_bl_count", count)
    local shard_label = shard_idx and (shard_idx - 1) or "?"
    ngx.log(ngx.INFO, "edge_blacklist_sync: ", count, " blocked IPs (ver=", new_ver, ", shard=", shard_label, ")")
    return true
end

return _M
