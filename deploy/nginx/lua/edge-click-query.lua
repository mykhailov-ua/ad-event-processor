-- GET /click campaign_id extraction from query string (strict UUID).
-- Runtime: nginx worker Lua VM; called from edge_track_policy.run_click.
--
-- Topology: edge :8180 gated by EDGE_EXPOSE_CLICK; tracker :8181-8184 always serves /click.
--
-- Args: ngx.req.get_uri_args(100) max 100 pairs; duplicate campaign_id rejected (table type).
--
-- Returns: normalized lowercase UUID or nil on parse failure.
--
-- Forbidden: body scan for campaign_id on GET /click; query-only per parser.mdc.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-click-query.lua
-- bash scripts/test/edge/lua_tests.sh
-- go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
local edge_uuid = require "edge-uuid"

local _M = {}

-- Strict UUID from query campaign_id only; duplicate key (table) rejected. Feeds slot_map CRC32C routing.
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
