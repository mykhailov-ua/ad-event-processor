-- ngx.balancer phase: weighted tracker peer with slot_map Redis-shard affinity fallback.
-- Runtime: all workers balance phase (balancer_by_lua_file); requires ngx.ctx.campaign_id from edge_track_policy.
-- No ngx.shared writes; reads slot_map and node_weights SHM only.
--
-- Consumers: nginx upstream trackers pool after access-check + edge_track_policy set campaign_id.
-- Peers: edge-tracker-peers.list unix_socket or host:port; healthcheck from init-worker spawn_checker.
--
-- Cache invalidation: none in this file (read-only SHM from edge-slot-map, edge-node-weights).
--
-- ngx.ctx outputs (request-scoped):
-- - campaign_id (string): input from edge_track_policy.
-- - redis_shard (number): slot_map.get_shard result when non-nil.
--
-- State machine (per balance):
-- - missing campaign_id -> return (nginx default peer).
-- - slot_map.get_shard(campaign_id) -> optional ngx.ctx.redis_shard.
-- - idx = node_weights.pick_peer_index(); if nil and shard set -> idx = tonumber(shard).
-- - idx nil -> return; else set_current_peer(peers.list[idx+1]) unix or TCP.
--
-- Invariant (must match Go StaticSlotSharder / sharding_amd64.s):
-- slot = CRC32C(edge_uuid.normalize_to_bytes(campaign_id)) & 1023; shard = slot_map s:{slot}.
--
-- Constants and limits:
-- - peer_idx = idx + 1 bounded 1..#peers.list; out of range logs WARN and skips set_current_peer.
--
-- Forbidden: Jump Hash; random peer when campaign_id present; CRC32 IEEE table; per-request control fetch.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-shard-balancer.lua
-- bash scripts/test/edge/lua_tests.sh
-- go test ./internal/domain/ -run Sharding -count=1
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
