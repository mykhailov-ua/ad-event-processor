-- Route exposure gates: GET /click and POST /openrtb/bid return 404 on edge when disabled.
-- Runtime: all workers access phase (access-check.lua after edge-ingress, before edge_track_policy).
-- Reads ngx.shared.edge_config via edge-config.get_flag with env fallback; no SHM writes.
--
-- Consumers: access-check.lua uri == /click or /openrtb/bid branches.
-- Edge listener :8180/:443 honors flags; tracker :8181-8184 serves /click regardless of edge_expose_click.
--
-- Cache invalidation: read edge_config flags mirrored by edge-config.sync (explicit replace on sync).
-- When get_flag nil: fail-open to env EDGE_EXPOSE_CLICK / EDGE_EXPOSE_OPENRTB truthy check.
--
-- ngx.shared edge_config flags (types):
-- - edge_expose_click (string|number|boolean serialized): truthy enables /click on edge.
-- - edge_expose_openrtb: truthy enables /openrtb/bid on edge.
--
-- State machine: click_enabled/openrtb_enabled -> require_* ngx.exit(404) when disabled.
-- truthy: 1, true, yes (case-insensitive string) or non-zero number.
--
-- Constants and limits: none beyond edge-config sync interval and env var names.
--
-- Forbidden: blocking /track, OPTIONS /track, or billing/settlement paths with these gates.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-route-gate.lua
-- bash scripts/test/edge/lua_tests.sh
local edge_config = require "edge-config"

local _M = {}

local function truthy(v)
    if v == nil then
        return false
    end
    if type(v) == "boolean" then
        return v
    end
    if type(v) == "number" then
        return v ~= 0
    end
    local s = string.lower(tostring(v))
    return s == "1" or s == "true" or s == "yes"
end

local function env_enabled(name)
    return truthy(os.getenv(name))
end

local function redis_enabled(field, env_name)
    local v = edge_config.get_flag(field)
    if v ~= nil then
        return truthy(v)
    end
    return env_enabled(env_name)
end

function _M.click_enabled()
    return redis_enabled("edge_expose_click", "EDGE_EXPOSE_CLICK")
end

function _M.openrtb_enabled()
    return redis_enabled("edge_expose_openrtb", "EDGE_EXPOSE_OPENRTB")
end

function _M.require_click()
    if not _M.click_enabled() then
        ngx.exit(ngx.HTTP_NOT_FOUND)
    end
end

function _M.require_openrtb()
    if not _M.openrtb_enabled() then
        ngx.exit(ngx.HTTP_NOT_FOUND)
    end
end

return _M
