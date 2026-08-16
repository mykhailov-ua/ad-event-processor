
local balancer = require "ngx.balancer"
local slot_map = require "edge-slot-map"
local node_weights = require "edge-node-weights"
local peers = require "edge-tracker-peers"

local campaign_id = ngx.ctx.campaign_id
if not campaign_id or campaign_id == "" then
    return
end

local shard = slot_map.get_shard(campaign_id)
if shard ~= nil then
    ngx.ctx.redis_shard = shard
end

local idx = node_weights.pick_peer_index()
if idx == nil and shard ~= nil then
    idx = tonumber(shard)
end
if idx == nil then
    return
end

local peer_idx = idx + 1
if peer_idx < 1 or peer_idx > #peers.list then
    ngx.log(ngx.WARN, "edge shard balancer: peer index out of range: ", idx)
    return
end

local peer = peers.list[peer_idx]
local ok, err
if peer.unix_socket then
    ok, err = balancer.set_current_peer(peer.unix_socket, 0)
else
    ok, err = balancer.set_current_peer(peer.host, peer.port)
end
if not ok then
    ngx.log(ngx.ERR, "edge shard balancer: set_current_peer failed: ", err or "unknown")
end
