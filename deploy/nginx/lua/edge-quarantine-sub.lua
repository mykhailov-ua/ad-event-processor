-- Redis pub/sub incremental blacklist stamps; complements periodic full sync in edge-blacklist-sync.
-- Runtime: worker 0 only (init-worker.lua quarantine_sub.start); ngx.thread for blocking I/O off request path.
--
-- Execution model:
-- - start() schedules ngx.timer.at(0, ...) on worker 0; timer callback ngx.thread.spawn(listen_loop).
-- - listen_loop runs in a light thread; main worker VM continues serving requests.
-- - red:read_reply blocks the light thread only; must not run subscribe/read_reply on the request thread.
--
-- Message path: fraud:quarantine -> apply_quarantine_message -> stamp_ips(..., false) (no _bl_ver bump).
-- Payload: JSON {"ips":[...]} or legacy plain IP string; empty payload triggers full sync() fallback.
--
-- Reconnect state machine (outer while not ngx.worker.exiting):
-- 1. connect_any_shard fail -> ERR log, ngx.sleep(2), retry.
-- 2. subscribe fail -> close, sleep(2), retry from connect.
-- 3. subscribed -> inner read_reply loop until read fail or worker exiting.
-- 4. read_reply fail -> WARN, break inner loop, close, outer loop reconnects from step 1.
--
-- Failure branches: spawn or initial timer schedule fail -> ERR log, no listener (incremental path dead; full sync timer still runs).
--
-- Forbidden: subscribe loop on every worker; blocking pub/sub on access-phase thread.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-quarantine-sub.lua
-- bash scripts/test/edge/lua_tests.sh
local blacklist_sync = require "edge-blacklist-sync"

local _M = {}

local CHANNEL = "fraud:quarantine"

local function listen_loop()
    while not ngx.worker.exiting() do
        local red, err = blacklist_sync.connect_any_shard()
        if not red then
            ngx.log(ngx.ERR, "edge_quarantine_sub: connect failed: ", err)
            ngx.sleep(2)
        else
            local res, sub_err = red:subscribe(CHANNEL)
            if not res then
                ngx.log(ngx.ERR, "edge_quarantine_sub: subscribe failed: ", sub_err)
                red:close()
                ngx.sleep(2)
            else
                ngx.log(ngx.INFO, "edge_quarantine_sub: subscribed to ", CHANNEL)
                while not ngx.worker.exiting() do
                    local reply, read_err = red:read_reply()
                    if not reply then
                        ngx.log(ngx.WARN, "edge_quarantine_sub: read_reply: ", read_err)
                        break
                    end
                    if type(reply) == "table" and reply[1] == "message" then
                        blacklist_sync.apply_quarantine_message(reply[2])
                    end
                end
                red:close()
            end
        end
    end
end

function _M.start()
    local ok, err = ngx.timer.at(0, function(premature)
        if premature then
            return
        end
        local spawn_ok, spawn_err = ngx.thread.spawn(listen_loop)
        if not spawn_ok then
            ngx.log(ngx.ERR, "edge_quarantine_sub: failed to spawn listener: ", spawn_err)
        end
    end)
    if not ok then
        ngx.log(ngx.ERR, "edge_quarantine_sub: failed to schedule listener: ", err)
    end
end

return _M
