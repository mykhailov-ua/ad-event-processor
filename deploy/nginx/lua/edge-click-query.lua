
local edge_uuid = require "edge-uuid"

local _M = {}

function _M.extract_campaign_id()
    local args, err = ngx.req.get_uri_args(100)
    if not args then
        ngx.log(ngx.ERR, "edge click query parse failed: ", err or "unknown")
        return nil
    end

    local cid = args.campaign_id
    if type(cid) == "table" then
        return nil
    end
    return edge_uuid.normalize(cid)
end

return _M
