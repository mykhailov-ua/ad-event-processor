-- Static tracker upstream peer table for ngx.balancer and node-weights sync.
-- Runtime: nginx worker; unix domain sockets to tracker processes.
--
-- Topology:
-- - tracker-0.sock -> logical :8181 / node_id tracker-1
-- - tracker-1.sock -> :8182 / tracker-2
-- - tracker-2.sock -> :8183 / tracker-3
-- - tracker-3.sock -> :8184 / tracker-4
--
-- Paths: /run/ad-event-processor/tracker/tracker-{0..3}.sock
--
-- Forbidden: editing peer list without matching deploy compose tracker replica count.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-tracker-peers.lua
-- bash scripts/test/edge/lua_tests.sh
local _M = {}

-- Balancer index mapping: pick_peer_index and slot_map shard return 0-based idx; list is 1-based (idx+1).
-- node_id strings (tracker-1..4) match control /ops/node-weights JSON; unix sockets to :8181-8184 trackers.
_M.list = {
    { unix_socket = "/run/ad-event-processor/tracker/tracker-0.sock", node_id = "tracker-1" },
    { unix_socket = "/run/ad-event-processor/tracker/tracker-1.sock", node_id = "tracker-2" },
    { unix_socket = "/run/ad-event-processor/tracker/tracker-2.sock", node_id = "tracker-3" },
    { unix_socket = "/run/ad-event-processor/tracker/tracker-3.sock", node_id = "tracker-4" },
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
