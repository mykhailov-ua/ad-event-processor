local edge_config = require "edge-config"

local _M = {}

function _M.is_whitelisted(asn)
    if not asn or asn == "" then
        return false
    end
    return edge_config.asn_whitelisted(asn)
end

return _M
