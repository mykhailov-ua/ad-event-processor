-- Control /ops/node-weights mirror into ngx.shared.node_weights for weighted tracker peer pick.
-- Runtime: worker 0 timer NODE_WEIGHTS_SYNC_INTERVAL_SEC default 10 s (init-worker.lua);
-- all workers read pick_peer_index() from edge-shard-balancer balance phase.
--
-- Consumers: edge-shard-balancer.lua pick_peer_index() before slot_map shard fallback.
-- Peers: edge-tracker-peers.lua unix sockets tracker-0..3 (:8181-8184 logical).
--
-- Cache invalidation: purge-then-replace on successful HTTP GET JSON sync.
-- Zero all w:{idx} for idx in 0..max(#peers.list-1, old_peer_count-1), write new weights, peer_count last.
-- epoch, epoch_lag, sync_ts updated before w:* purge so stale() stays accurate during replace.
-- Stale detection: TTL wall-clock on sync_ts (> 2x interval) or epoch_lag > STALE_EPOCH_LAG.
-- Sync HTTP fail: fail-open on prior weights until stale threshold; then equal-weight or drain freeze.
--
-- ngx.shared node_weights (types):
-- - sync_interval (number): copy of SYNC_INTERVAL_SEC stamped each sync.
-- - epoch (number): control routing epoch from JSON doc.epoch.
-- - epoch_lag (number): doc.epoch_lag; stale when > STALE_EPOCH_LAG.
-- - sync_ts (number unix s): last successful sync wall time.
-- - peer_count (number): active peer indices 0..n-1.
-- - w:{idx} (number): fixed-point weight floor(weight * WEIGHT_SCALE); 0 means drained peer.
--
-- State machine:
-- - Cold boot: worker 0 timer at 0 -> sync(); pick_peer_index nil until peer_count > 0.
-- - Fresh: weighted random by w:* cumulatives; roll from crc32_short(request_id:now) % WEIGHT_SCALE.
-- - Stale + CONTROL_FAIL_OPEN unset: equal WEIGHT_SCALE per peer (drain freeze semantics off for pick).
-- - Stale + fail_open false: drain_frozen true; pick still runs equal-weight when stale() in pick loop.
-- - total weight <= 0: crc32_short fallback % n without weight table.
--
-- Constants and limits:
-- - SYNC_INTERVAL_SEC default 10 (NODE_WEIGHTS_SYNC_INTERVAL_SEC env).
-- - STALE_EPOCH_LAG 2; stale wall clock = sync_ts older than SYNC_INTERVAL_SEC * STALE_EPOCH_LAG.
-- - WEIGHT_SCALE 1000000; CONTROL_FAIL_OPEN env 1|true|TRUE skips drain_frozen stale gate.
-- - CONTROL_URL default unix:///run/ad-event-processor/control/http.sock; HTTP via edge-net.http_get_json.
--
-- Forbidden: picking peer without slot_map campaign affinity fallback in edge-shard-balancer.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-node-weights.lua
-- bash scripts/test/edge/lua_tests.sh
local _M = {}

local dict = ngx.shared.node_weights
local peers = require "edge-tracker-peers"
local edge_net = require "edge-net"

local getenv = os.getenv

local CONTROL_URL = os.getenv "CONTROL_URL"
    or os.getenv "MANAGEMENT_URL"
    or "unix:///run/ad-event-processor/control/http.sock"
local SYNC_INTERVAL_SEC = tonumber(os.getenv "NODE_WEIGHTS_SYNC_INTERVAL_SEC" or "") or 10
local STALE_EPOCH_LAG = 2
local WEIGHT_SCALE = 1000000

function _M.sync_interval_sec()
    return SYNC_INTERVAL_SEC
end

function _M.fail_open()
    local raw = getenv "CONTROL_FAIL_OPEN" or "0"
    return raw == "1" or raw == "true" or raw == "TRUE"
end

function _M.set_getenv_for_test(fn)
    getenv = fn or os.getenv
end

function _M.epoch()
    return dict:get "epoch" or 0
end

function _M.epoch_lag()
    return dict:get "epoch_lag" or 0
end

function _M.stale()
    local sync_ts = dict:get "sync_ts"
    if not sync_ts then
        return true
    end
    if ngx.time() - sync_ts > SYNC_INTERVAL_SEC * STALE_EPOCH_LAG then
        return true
    end
    local lag = dict:get "epoch_lag" or 0
    return lag > STALE_EPOCH_LAG
end

function _M.drain_frozen()
    if _M.fail_open() then
        return false
    end
    return _M.stale()
end

-- Weighted peer pick: crc32_short(request_id:now) drives roll; stale map -> equal WEIGHT_SCALE per peer
-- unless CONTROL_FAIL_OPEN. total<=0 falls back to crc32_short % n. Returns 0-based idx for balancer.
function _M.pick_peer_index()
    local n = dict:get "peer_count" or 0
    if n <= 0 then
        return nil
    end
    if n == 1 then
        return 0
    end

    local equal = _M.stale() and not _M.fail_open()
    local total = 0
    local cum = {}
    for i = 0, n - 1 do
        local w
        if equal then
            w = WEIGHT_SCALE
        else
            w = dict:get("w:" .. i) or 0
        end
        if w < 0 then
            w = 0
        end
        total = total + w
        cum[#cum + 1] = total
    end
    if total <= 0 then
        local key = (ngx.var.request_id or "") .. ":" .. tostring(ngx.now())
        return ngx.crc32_short(key) % n
    end

    local key = (ngx.var.request_id or "") .. ":" .. tostring(ngx.now())
    local roll = ngx.crc32_short(key) % WEIGHT_SCALE
    local target = (roll / WEIGHT_SCALE) * total
    for i = 1, #cum do
        if target < cum[i] then
            return i - 1
        end
    end
    return n - 1
end

-- Purge-then-replace: zero w:0..n-1, write new weights, peer_count last. epoch/epoch_lag from control JSON.
function _M.sync()
    dict:set("sync_interval", SYNC_INTERVAL_SEC)
    local url = CONTROL_URL .. "/ops/node-weights"
    local doc, err = edge_net.http_get_json(url)
    if not doc then
        ngx.log(ngx.WARN, "edge node weights sync failed: ", err or "unknown")
        return
    end

    local nodes = doc.node_weights
    if not nodes or type(nodes) ~= "table" then
        ngx.log(ngx.WARN, "edge node weights sync: missing node_weights array")
        return
    end

    dict:set("epoch", doc.epoch or 0)
    dict:set("epoch_lag", doc.epoch_lag or 0)
    dict:set("sync_ts", ngx.time())

    local old_peer_count = dict:get "peer_count" or 0
    local purge_n = #peers.list
    if old_peer_count > purge_n then
        purge_n = old_peer_count
    end
    for i = 0, purge_n - 1 do
        dict:set("w:" .. i, 0)
    end

    local max_idx = -1
    for _, node in ipairs(nodes) do
        local idx = node.peer_index
        if idx == nil and node.node_id then
            idx = peers.index_for_node_id(node.node_id)
        end
        if idx ~= nil and idx >= 0 and idx < #peers.list then
            local w = tonumber(node.weight) or 0
            if w < 0 then
                w = 0
            end
            dict:set("w:" .. idx, math.floor(w * WEIGHT_SCALE + 0.5))
            if idx > max_idx then
                max_idx = idx
            end
        end
    end

    local peer_count = max_idx + 1
    if peer_count < 1 then
        peer_count = #nodes
    end
    if peer_count > #peers.list then
        peer_count = #peers.list
    end
    dict:set("peer_count", peer_count)
end

return _M
