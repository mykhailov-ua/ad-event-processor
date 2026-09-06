-- Role: edge-config ASN whitelist invalidation on Redis sync failure (asn_whitelist_sync_failclosed).
-- Execution context: worker 0 sync timer; generational _asn_ver stamps.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local config_store = {}

local function make_dict(store)
    return {
        get = function(_, key)
            return store[key]
        end,
        set = function(_, key, val)
            store[key] = val
            return true
        end,
        delete = function(_, key)
            store[key] = nil
        end,
    }
end

ngx = {
    log = function() end,
    shared = {
        edge_config = make_dict(config_store),
    },
}

package.loaded["edge-blacklist-sync"] = {
    connect_any_shard = function()
        return nil, "down"
    end,
}

package.loaded["edge-circuit"] = {
    record_err = function() end,
}

package.loaded["edge-config"] = nil
local edge_config = require "edge-config"

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: ", msg, "\n")
    end
end

config_store["_asn_ver"] = 3
config_store["asn_cdn:15169"] = 3

edge_config.sync()

assert_true(config_store["_asn_ver"] == 4, "sync fail bumps _asn_ver")
assert_true(not edge_config.asn_whitelisted "15169", "stale ASN stamp fails closed after sync fail")

io.write(string.format("asn_sync_failclosed_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
