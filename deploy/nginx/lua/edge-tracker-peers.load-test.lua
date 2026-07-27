-- edge-tracker-peers.lua: constrained load-test profile (tracker-0/1 only).
-- Mounted over edge-tracker-peers.lua by docker-compose.load-test.yaml.

local _M = {}

_M.list = {
    { host = "127.0.0.1", port = 8181, node_id = "tracker-1" },
    { host = "127.0.0.1", port = 8182, node_id = "tracker-2" },
}

function _M.index_for_node_id(node_id)
    if not node_id or node_id == "" then
        return nil
    end
    for i, peer in ipairs(_M.list) do
        if peer.node_id == node_id then
            return i - 1
        end
    end
    return nil
end

return _M
