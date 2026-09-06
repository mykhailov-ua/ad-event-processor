-- ASN perimeter bypass: CDN/mobile ASNs skip IP blacklist lookup (not fraud scoring).
-- Runtime: nginx worker Lua VM; reads ngx.shared.edge_config via edge-config.
--
-- Topology: trusted ASN from EDGE_TRUSTED_ASN_HEADER (default X-Edge-Trusted-ASN) only.
-- Client X-Client-ASN is cleared in access-check before blacklist (client_asn_blacklist_bypass); not read for bypass.
-- CDN operators: inject X-Edge-Trusted-ASN at the hop before this nginx, or set EDGE_TRUSTED_ASN_HEADER
-- to the header name your CDN sets when nginx is not reachable by end clients.
--
-- ngx.shared edge_config keys:
-- - _asn_ver (number): active generation from edge-config.sync.
-- - asn_cdn:{asn}, asn_mobile:{asn} (number): generation stamp; whitelisted when stamp == _asn_ver.
--
-- Returns: true when ASN stamp matches _asn_ver (CDN or mobile); false when stale, missing, or empty ASN.
--
-- Forbidden: treating ASN whitelist as fraud pass on tracker; reading http_x_client_asn on public edge.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-asn.lua
-- bash scripts/test/edge/lua_tests.sh unit
local edge_config = require "edge-config"

local _M = {}

local getenv = os.getenv

local DEFAULT_TRUSTED_ASN_HEADER = "X-Edge-Trusted-ASN"
local UNTRUSTED_ASN_HEADER = "X-Client-ASN"

local function reload_trusted_header()
    local name = getenv "EDGE_TRUSTED_ASN_HEADER" or DEFAULT_TRUSTED_ASN_HEADER
    if name == "" then
        name = DEFAULT_TRUSTED_ASN_HEADER
    end
    return name
end

local TRUSTED_ASN_HEADER = reload_trusted_header()

local function header_http_var(name)
    name = string.lower(name)
    name = string.gsub(name, "-", "_")
    return "http_" .. name
end

local TRUSTED_ASN_VAR = header_http_var(TRUSTED_ASN_HEADER)

function _M.set_getenv_for_test(fn)
    if fn then
        getenv = fn
    else
        getenv = os.getenv
    end
    TRUSTED_ASN_HEADER = reload_trusted_header()
    TRUSTED_ASN_VAR = header_http_var(TRUSTED_ASN_HEADER)
end

function _M.trusted_header_name()
    return TRUSTED_ASN_HEADER
end

function _M.trusted_client_asn()
    return ngx.var[TRUSTED_ASN_VAR]
end

-- Drop client-spoofable X-Client-ASN unless operator explicitly configured it as trusted header name.
function _M.sanitize_untrusted_asn_headers()
    if TRUSTED_ASN_HEADER ~= UNTRUSTED_ASN_HEADER then
        ngx.req.clear_header(UNTRUSTED_ASN_HEADER)
    end
end

function _M.is_whitelisted(asn)
    if not asn or asn == "" then
        return false
    end
    return edge_config.asn_whitelisted(asn)
end

return _M
