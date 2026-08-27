local hc = require "resty.upstream.healthcheck"
local edge_config = require "edge-config"
local blacklist_sync = require "edge-blacklist-sync"
local edge_slot_map = require "edge-slot-map"
local edge_node_weights = require "edge-node-weights"
local quarantine_sub = require "edge-quarantine-sub"
local tcp_fp_sync = require "edge-tcp-fp-sync"

if ngx.worker.id() ~= 0 then
    return
end

local CONFIG_SYNC_INTERVAL = 5
local BLACKLIST_SYNC_INTERVAL = tonumber(os.getenv("EDGE_BLACKLIST_SYNC_INTERVAL_SEC") or "") or 5
local BLACKLIST_CHANGELOG_DRAIN_INTERVAL = tonumber(os.getenv("EDGE_BLACKLIST_CHANGELOG_DRAIN_SEC") or "") or 1
local SLOT_MAP_SYNC_INTERVAL = tonumber(os.getenv("SLOT_MAP_SYNC_INTERVAL_SEC") or "") or 10
local NODE_WEIGHTS_SYNC_INTERVAL = tonumber(os.getenv("NODE_WEIGHTS_SYNC_INTERVAL_SEC") or "") or 10
local TCP_FP_SYNC_INTERVAL = tonumber(os.getenv("TCP_FP_SYNC_INTERVAL_SEC") or "") or 2

local function sync_edge_config(premature)
    if premature then
        return
    end
    edge_config.sync()
    local ok, timer_err = ngx.timer.at(CONFIG_SYNC_INTERVAL, sync_edge_config)
    if not ok then
        ngx.log(ngx.ERR, "failed to reschedule edge config sync: ", timer_err)
    end
end

local function sync_blacklist(premature)
    if premature then
        return
    end
    blacklist_sync.sync()
    local ok, timer_err = ngx.timer.at(BLACKLIST_SYNC_INTERVAL, sync_blacklist)
    if not ok then
        ngx.log(ngx.ERR, "failed to reschedule blacklist sync: ", timer_err)
    end
end

local timer_ok, timer_err = ngx.timer.at(0, sync_edge_config)
if not timer_ok then
    ngx.log(ngx.ERR, "failed to start edge config sync: ", timer_err)
end

timer_ok, timer_err = ngx.timer.at(0, sync_blacklist)
if not timer_ok then
    ngx.log(ngx.ERR, "failed to start blacklist sync: ", timer_err)
end

local function drain_blacklist_changelog(premature)
    if premature then
        return
    end
    local n = blacklist_sync.drain_pending_changelog()
    if n > 0 then
        ngx.log(ngx.INFO, "edge_blacklist_sync: drained ", n, " pending changelog IPs")
    end
    local ok, schedule_err = ngx.timer.at(BLACKLIST_CHANGELOG_DRAIN_INTERVAL, drain_blacklist_changelog)
    if not ok then
        ngx.log(ngx.ERR, "failed to reschedule blacklist changelog drain: ", schedule_err)
    end
end

timer_ok, timer_err = ngx.timer.at(BLACKLIST_CHANGELOG_DRAIN_INTERVAL, drain_blacklist_changelog)
if not timer_ok then
    ngx.log(ngx.ERR, "failed to start blacklist changelog drain: ", timer_err)
end

quarantine_sub.start()

local function sync_slot_map(premature)
    if premature then
        return
    end
    edge_slot_map.sync()
    local ok, err = ngx.timer.at(SLOT_MAP_SYNC_INTERVAL, sync_slot_map)
    if not ok then
        ngx.log(ngx.ERR, "failed to reschedule slot map sync: ", err)
    end
end

timer_ok, timer_err = ngx.timer.at(0, sync_slot_map)
if not timer_ok then
    ngx.log(ngx.ERR, "failed to start slot map sync: ", timer_err)
end

local function sync_node_weights(premature)
    if premature then
        return
    end
    edge_node_weights.sync()
    local ok, err = ngx.timer.at(NODE_WEIGHTS_SYNC_INTERVAL, sync_node_weights)
    if not ok then
        ngx.log(ngx.ERR, "failed to reschedule node weights sync: ", err)
    end
end

timer_ok, timer_err = ngx.timer.at(0, sync_node_weights)
if not timer_ok then
    ngx.log(ngx.ERR, "failed to start node weights sync: ", timer_err)
end

local function sync_tcp_fp(premature)
    if premature then
        return
    end
    local ok, err = tcp_fp_sync.sync()
    if not ok and err then
        ngx.log(ngx.WARN, "edge_tcp_fp_sync: ", err)
    end
    local ok2, resched_err = ngx.timer.at(TCP_FP_SYNC_INTERVAL, sync_tcp_fp)
    if not ok2 then
        ngx.log(ngx.ERR, "failed to reschedule tcp fp sync: ", resched_err)
    end
end

timer_ok, timer_err = ngx.timer.at(0, sync_tcp_fp)
if not timer_ok then
    ngx.log(ngx.ERR, "failed to start tcp fp sync: ", timer_err)
end

local ok, err = hc.spawn_checker({
    shm = "healthcheck",
    upstream = "trackers",
    type = "http",
    http_req = "GET /health HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n",
    interval = 2000,
    timeout = 1000,
    fall = 2,
    rise = 2,
    valid_statuses = { 200 },
    concurrency = 4,
})
if not ok then
    ngx.log(ngx.ERR, "failed to spawn upstream health checker: ", err)
end
