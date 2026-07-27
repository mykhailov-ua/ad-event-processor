-- edge-node-weights.lua: polls management /ops/node-weights for capacity-weighted tracker pick (M5.2).
-- Stale sync or epoch_lag > 2 → equal weights (freeze drain). H6: used only from balancer_by_lua (new connections).

local _M = {}

local dict = ngx.shared.node_weights
local peers = require "edge-tracker-peers"

local getenv = os.getenv

local MANAGEMENT_URL = os.getenv("MANAGEMENT_URL") or "http://127.0.0.1:8188"
local SYNC_INTERVAL_SEC = tonumber(os.getenv("NODE_WEIGHTS_SYNC_INTERVAL_SEC") or "") or 10
local STALE_EPOCH_LAG = 2
local WEIGHT_SCALE = 1000000

local function parse_url(url)
    local host, port, path = string.match(url, "^https?://([^:/]+):?(%d*)(/.*)$")
    if not host then
        host, path = string.match(url, "^https?://([^/]+)(/.*)$")
        port = ""
    end
    if not host then
        return nil
    end
    if port == "" or port == nil then
        port = 8188
    else
        port = tonumber(port)
    end
    if not path or path == "" then
        path = "/"
    end
    return host, port, path
end

local function http_get_json(url)
    local host, port, path = parse_url(url)
    if not host then
        return nil, "invalid management url"
    end
    local sock = ngx.socket.tcp()
    sock:settimeout(2000)
    local ok, err = sock:connect(host, port)
    if not ok then
        return nil, err
    end
    local req = "GET " .. path .. " HTTP/1.1\r\nHost: " .. host .. "\r\nConnection: close\r\nAccept: application/json\r\n\r\n"
    local sent, send_err = sock:send(req)
    if not sent then
        sock:close()
        return nil, send_err
    end
    local data, read_err = sock:receive("*a")
    sock:close()
    if not data then
        return nil, read_err
    end
    local body = string.match(data, "\r\n\r\n(.*)$")
    if not body then
        return nil, "empty http body"
    end
    local cjson = require "cjson.safe"
    return cjson.decode(body)
end

function _M.sync_interval_sec()
    return SYNC_INTERVAL_SEC
end

function _M.fail_open()
    local raw = getenv("CONTROL_FAIL_OPEN") or "0"
    return raw == "1" or raw == "true" or raw == "TRUE"
end

-- set_getenv_for_test overrides getenv in offline unit tests only.
function _M.set_getenv_for_test(fn)
    getenv = fn or os.getenv
end

function _M.epoch()
    return dict:get("epoch") or 0
end

function _M.epoch_lag()
    return dict:get("epoch_lag") or 0
end

function _M.stale()
    local sync_ts = dict:get("sync_ts")
    if not sync_ts then
        return true
    end
    if ngx.time() - sync_ts > SYNC_INTERVAL_SEC * STALE_EPOCH_LAG then
        return true
    end
    local lag = dict:get("epoch_lag") or 0
    return lag > STALE_EPOCH_LAG
end

function _M.drain_frozen()
    if _M.fail_open() then
        return false
    end
    return _M.stale()
end

-- pick_peer_index returns 0-based tracker peer index using weighted random over the epoch snapshot.
function _M.pick_peer_index()
    local n = dict:get("peer_count") or 0
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

function _M.sync()
    dict:set("sync_interval", SYNC_INTERVAL_SEC)
    local url = MANAGEMENT_URL .. "/ops/node-weights"
    local doc, err = http_get_json(url)
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
