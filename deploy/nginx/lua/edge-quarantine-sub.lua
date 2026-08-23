
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
    local ok, err = ngx.thread.spawn(listen_loop)
    if not ok then
        ngx.log(ngx.ERR, "edge_quarantine_sub: failed to spawn listener: ", err)
    end
end

return _M
