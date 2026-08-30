-- Stage XDP-written TCP SYN fingerprints from Redis into ngx.shared.tcp_fp_cache for L7 header forward.
-- Runtime: worker 0 timer TCP_FP_SYNC_INTERVAL_SEC default 2 s (init-worker.lua); connect via edge-blacklist-sync.
--
-- Consumer: edge-ingress.lua reads tcp_fp_cache on access phase -> X-TCP-MSS/TTL/WINDOW/SIG upstream headers.
--
-- Redis keys (shard 0 staging, written by edge-xdp):
-- - edge:tcp_fp:recent ZSET; member {ip}:{tcp_hash_hex}; ZREVRANGE 0 511 picks hot IPs.
-- - edge:tcp_fp:ip:{ip} HASH fields ttl, window, mss, tcp_hash (Redis TTL 1 h).
--
-- ngx.shared tcp_fp_cache mapping (each set TTL 3600 s):
-- - {ip}           -> mss (0..255)
-- - t:{ip}         -> ttl (0..255)
-- - w:{ip}         -> window (0..65535)
-- - h:{ip}         -> tcp_hash 8-char hex sig
--
-- sync pipeline: ZREVRANGE recent -> parse ip from member -> HMGET pipeline per ip -> validate ranges -> cache:set.
-- Empty recent set: return true (no SHM mutation).
--
-- Failure branches (prior tcp_fp_cache entries retained):
-- - tcp_fp_cache dict missing: return false.
-- - connect_any_shard fail: return false.
-- - ZREVRANGE or pipeline fail: return false; init-worker logs WARN, reschedules timer.
--
-- Forbidden: Redis read in access phase; claiming tcp_fp replaces tracker FilterEngine or blocks all fraud.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-tcp-fp-sync.lua
-- bash scripts/test/edge/lua_tests.sh
-- go test ./internal/edge/... -count=1
local blacklist_sync = require "edge-blacklist-sync"

local _M = {}

local cache = ngx.shared.tcp_fp_cache
local RECENT_KEY = "edge:tcp_fp:recent"
local IP_KEY_PREFIX = "edge:tcp_fp:ip:"

function _M.sync()
    if not cache then
        return false, "tcp_fp_cache missing"
    end

    local red, err = blacklist_sync.connect_any_shard()
    if not red then
        return false, err
    end

    local members, zerr = red:zrevrange(RECENT_KEY, 0, 511)
    if not members then
        red:set_keepalive(10000, 100)
        return false, zerr
    end

    local ips = {}
    for _, member in ipairs(members) do
        local ip = member:match "^([^:]+)"
        if ip then
            ips[#ips + 1] = ip
        end
    end

    if #ips == 0 then
        red:set_keepalive(10000, 100)
        return true
    end

    red:init_pipeline()
    for _, ip in ipairs(ips) do
        red:hmget(IP_KEY_PREFIX .. ip, "ttl", "window", "mss", "tcp_hash")
    end
    local results, perr = red:commit_pipeline()
    red:set_keepalive(10000, 100)
    if not results then
        return false, perr
    end

    local stamped = 0
    for i, ip in ipairs(ips) do
        local res = results[i]
        local ttl, win, mss, tcp_hash
        if type(res) == "table" then
            ttl, win, mss, tcp_hash = res[1], res[2], res[3], res[4]
        else
            ttl, win, mss, tcp_hash = res, nil, nil, nil
        end
        if mss and mss ~= ngx.null then
            local n = tonumber(mss)
            if n and n >= 0 and n <= 255 then
                cache:set(ip, n, 3600)
                stamped = stamped + 1
            end
        end
        if ttl and ttl ~= ngx.null then
            local t = tonumber(ttl)
            if t and t >= 0 and t <= 255 then
                cache:set("t:" .. ip, t, 3600)
            end
        end
        if win and win ~= ngx.null then
            local w = tonumber(win)
            if w and w >= 0 and w <= 65535 then
                cache:set("w:" .. ip, w, 3600)
            end
        end
        if tcp_hash and tcp_hash ~= ngx.null and type(tcp_hash) == "string" and #tcp_hash == 8 then
            cache:set("h:" .. ip, tcp_hash, 3600)
        end
    end

    if stamped > 0 then
        ngx.log(ngx.INFO, "edge_tcp_fp_sync: stamped ", stamped, " TCP fp entries")
    end
    return true
end

return _M
