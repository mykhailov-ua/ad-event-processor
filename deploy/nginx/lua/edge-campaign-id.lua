-- Trusted campaign_id header for stream/peek fallbacks when body DFA does not extract an id.
-- Runtime: nginx worker Lua VM; reads trusted header only after clearing client X-Campaign-Id.
--
-- Topology: EDGE_TRUSTED_CAMPAIGN_ID_HEADER (default X-Edge-Trusted-Campaign-Id) for CDN inner hops.
-- Client X-Campaign-Id cleared before read (header_campaign_id_trusted_only); not used for ngx.ctx routing or edge_rl.
-- Body-extracted id from edge-parse-dfa wins when present; trusted header is fallback only.
--
-- Returns: lowercase UUID via edge-uuid.normalize; nil when missing or invalid.
--
-- Forbidden: reading http_x_campaign_id on public edge; routing from unvalidated header strings.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-campaign-id.lua
-- bash scripts/test/edge/lua_tests.sh unit
local edge_uuid = require "edge-uuid"
local edge_parse_dfa = require "edge-parse-dfa"

local _M = {}

local getenv = os.getenv

local DEFAULT_TRUSTED_CAMPAIGN_ID_HEADER = "X-Edge-Trusted-Campaign-Id"
local UNTRUSTED_CAMPAIGN_ID_HEADER = "X-Campaign-Id"

local function reload_trusted_header()
    local name = getenv "EDGE_TRUSTED_CAMPAIGN_ID_HEADER" or DEFAULT_TRUSTED_CAMPAIGN_ID_HEADER
    if name == "" then
        name = DEFAULT_TRUSTED_CAMPAIGN_ID_HEADER
    end
    return name
end

local TRUSTED_CAMPAIGN_ID_HEADER = reload_trusted_header()

local function header_http_var(name)
    name = string.lower(name)
    name = string.gsub(name, "-", "_")
    return "http_" .. name
end

local TRUSTED_CAMPAIGN_ID_VAR = header_http_var(TRUSTED_CAMPAIGN_ID_HEADER)

function _M.set_getenv_for_test(fn)
    if fn then
        getenv = fn
    else
        getenv = os.getenv
    end
    TRUSTED_CAMPAIGN_ID_HEADER = reload_trusted_header()
    TRUSTED_CAMPAIGN_ID_VAR = header_http_var(TRUSTED_CAMPAIGN_ID_HEADER)
end

function _M.trusted_header_name()
    return TRUSTED_CAMPAIGN_ID_HEADER
end

function _M.sanitize_untrusted_campaign_id_headers()
    if TRUSTED_CAMPAIGN_ID_HEADER ~= UNTRUSTED_CAMPAIGN_ID_HEADER then
        ngx.req.clear_header(UNTRUSTED_CAMPAIGN_ID_HEADER)
    end
end

function _M.trusted_campaign_id()
    local raw = ngx.var[TRUSTED_CAMPAIGN_ID_VAR]
    if not raw or raw == "" then
        return nil
    end
    return edge_uuid.normalize(raw)
end

-- Body DFA id is authoritative; trusted header used only when body_id absent.
function _M.resolve_campaign_id(body_id)
    _M.sanitize_untrusted_campaign_id_headers()
    if body_id and body_id ~= "" then
        return edge_uuid.normalize(body_id)
    end
    return _M.trusted_campaign_id()
end

-- campaign_id_scan_evasion: DFA nil,nil at scan/CL cap without trusted header must not use IP-only edge_rl fallback.
function _M.scan_evasion(body, body_id, perr, cl)
    if body_id ~= nil or perr ~= nil then
        return false
    end
    local max_scan = edge_parse_dfa.MAX_SCAN_BYTES
    local body_len = body and #body or 0
    if (cl and cl > max_scan) or body_len > max_scan then
        return true
    end
    if body_len >= max_scan or (cl and cl >= max_scan) then
        return true
    end
    return false
end

return _M
