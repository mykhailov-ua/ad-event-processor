-- ASN perimeter bypass: CDN/mobile ASNs skip IP blacklist lookup (not fraud scoring).
-- Runtime: nginx worker Lua VM; reads ngx.shared.edge_config via edge-config.
--
-- Topology: X-Client-ASN from edge/CDN header; whitelist loaded from Redis config:values (worker 0 sync).
--
-- ngx.shared edge_config keys:
-- - _asn_ver (number): active generation from edge-config.sync.
-- - asn_cdn:{asn}, asn_mobile:{asn} (number): generation stamp; whitelisted when stamp == _asn_ver.
--
-- Returns: true when ASN stamp matches _asn_ver (CDN or mobile); false when stale, missing, or empty ASN.
--
-- Forbidden: treating ASN whitelist as fraud pass on tracker; tracker runs full FilterEngine.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-asn.lua
-- bash scripts/test/edge/lua_tests.sh
local edge_config = require "edge-config"

local _M = {}

function _M.is_whitelisted(asn)
    if not asn or asn == "" then
        return false
    end
    return edge_config.asn_whitelisted(asn)
end

return _M
