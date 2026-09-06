-- Role: edge-blacklist-sync drain_pending_changelog restore on stamp_ips failure (pending_drain_restore_on_fail).
-- Execution context: init-worker timer drains _bl_pending via stamp_ips(..., false).
-- Invariants proved: pending snapshot restored when stamp_ips returns false.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local cache_store = {}
local metric_store = {}

ngx = {
    WARN = 1,
    INFO = 2,
    ERR = 3,
    log = function() end,
    time = function()
        return 1
    end,
    shared = {
        blacklist_cache = {
            get = function(_, k)
                return cache_store[k]
            end,
            set = function(_, k, v)
                cache_store[k] = v
            end,
            delete = function(_, k)
                cache_store[k] = nil
            end,
        },
        sentinel_cache = {
            get = function()
                return nil
            end,
        },
        circuit_breaker = {
            incr = function(_, _, delta, init)
                return (init or 0) + delta
            end,
        },
        edge_metrics = {
            incr = function(_, key, delta, init)
                metric_store[key] = (metric_store[key] or init or 0) + delta
                return metric_store[key]
            end,
        },
    },
}

package.loaded["resty.redis"] = {}
package.loaded["edge-circuit"] = {
    record_err = function() end,
}
package.loaded["edge-metrics"] = nil
package.loaded["edge-blacklist-sync"] = nil

local blacklist_sync = require "edge-blacklist-sync"

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: ", msg, "\n")
    end
end

local snapshot = "10.0.0.1\n10.0.0.2\n10.0.0.3\n"
cache_store["_bl_pending"] = snapshot
cache_store["_bl_ver"] = 1

blacklist_sync.set_stamp_ips_fail_for_test(true)
assert_true(blacklist_sync.drain_pending_changelog() == 0, "drain returns 0 on stamp failure")
assert_true(cache_store["_bl_pending"] == snapshot, "pending queue restored from snapshot")
assert_true(cache_store["b:10.0.0.1"] == nil, "no partial stamp on failure")

blacklist_sync.set_stamp_ips_fail_for_test(false)
assert_true(blacklist_sync.drain_pending_changelog() > 0, "drain succeeds after failure cleared")
assert_true(cache_store["b:10.0.0.1"] == 1, "first pending IP stamped")
assert_true(cache_store["b:10.0.0.2"] == 1, "second pending IP stamped")

blacklist_sync.reset_test_hooks()

io.write(string.format("bl_pending_drain_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
