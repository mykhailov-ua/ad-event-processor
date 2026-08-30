-- Optional edge tarpit: bounded ngx.sleep delay for abusive header/body shapes at L7 perimeter.
-- Runtime: all workers access phase (access-check.lua after perimeter_gate, before edge-ingress).
-- No ngx.shared; config from env at module load (reload_config); edge-metrics tarpit counters on delay.
--
-- Consumers: access-check.lua edge_tarpit.maybe_delay(); compliance.mdc caps duration on edge only.
--
-- Cache invalidation: none (env-driven ENABLED flag; set_getenv_for_test for lua_tests tarpit).
--
-- State machine:
-- - EDGE_TARPIT_ENABLED false (default) -> maybe_delay no-op.
-- - enabled: count request headers; read content_length; compute_delay -> ngx.sleep if delay > 0.
--
-- Constants and limits:
-- - EDGE_TARPIT_MAX_HEADERS default 64.
-- - EDGE_TARPIT_BODY_BYTES default 65536.
-- - EDGE_TARPIT_MAX_SEC default 2; hard cap 15 s; negative clamped to 0.
-- - Header delay: min(MAX_SEC, 0.25 + (headers - MAX_HEADERS) * 0.05).
-- - Body delay: min(MAX_SEC, 0.5 + (content_length - MAX_BODY) / MAX_BODY); max of header/body terms wins.
--
-- Failure modes: disabled -> no-op; enabled -> adds latency only (request still proceeds to tracker).
--
-- Forbidden: unbounded sleep; tarpit on tracker unix upstream sockets or billing/settlement paths.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-tarpit.lua
-- bash scripts/test/edge/lua_tests.sh tarpit
local edge_metrics = require "edge-metrics"

local _M = {}

local getenv = os.getenv

local ENABLED = false
local MAX_HEADERS = 64
local MAX_BODY = 65536
local MAX_SEC = 2

local function env_bool(name, default)
    local v = getenv(name)
    if v == nil or v == "" then
        return default
    end
    v = string.lower(v)
    return v == "1" or v == "true" or v == "yes" or v == "on"
end

local function env_num(name, default)
    local v = tonumber(getenv(name) or "")
    if not v then
        return default
    end
    return v
end

local function reload_config()
    ENABLED = env_bool("EDGE_TARPIT_ENABLED", false)
    MAX_HEADERS = env_num("EDGE_TARPIT_MAX_HEADERS", 64)
    MAX_BODY = env_num("EDGE_TARPIT_BODY_BYTES", 65536)
    MAX_SEC = env_num("EDGE_TARPIT_MAX_SEC", 2)
    if MAX_SEC > 15 then
        MAX_SEC = 15
    end
    if MAX_SEC < 0 then
        MAX_SEC = 0
    end
end

reload_config()

function _M.set_getenv_for_test(fn)
    if fn then
        getenv = fn
    else
        getenv = os.getenv
    end
    reload_config()
end

function _M.compute_delay(header_count, content_length)
    local delay = 0
    if header_count > MAX_HEADERS then
        delay = math.min(MAX_SEC, 0.25 + (header_count - MAX_HEADERS) * 0.05)
    end
    if content_length > MAX_BODY then
        local body_delay = math.min(MAX_SEC, 0.5 + (content_length - MAX_BODY) / MAX_BODY)
        if body_delay > delay then
            delay = body_delay
        end
    end
    return delay
end

function _M.enabled()
    return ENABLED
end

function _M.maybe_delay()
    if not ENABLED then
        return
    end

    local headers = ngx.req.get_headers()
    local n = 0
    for _ in pairs(headers) do
        n = n + 1
    end

    local cl = tonumber(ngx.var.content_length) or 0
    local delay = _M.compute_delay(n, cl)

    if delay > 0 then
        edge_metrics.record_tarpit(delay)
        ngx.sleep(delay)
    end
end

return _M
