-- Role: edge-blacklist-sync _bl_pending byte cap.
-- Execution context: stamp_ips overflow tail -> append_pending_ips with PENDING_MAX_BYTES ceiling.
-- Invariants proved: pending string bounded; overflow IPs immediate-stamped at _bl_ver (not dropped).
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local cache_store = {}
local metric_store = {}
local circuit_errs = 0

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
    record_err = function()
        circuit_errs = circuit_errs + 1
    end,
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

blacklist_sync.set_env_for_test(function(name)
    if name == "EDGE_BLACKLIST_PENDING_MAX_BYTES" then
        return "128"
    end
    if name == "EDGE_BLACKLIST_CHANGELOG_MAX_IPS" then
        return "2"
    end
    return nil
end)

cache_store["_bl_ver"] = 1
local batch = {}
for i = 1, 20 do
    batch[i] = string.format("10.0.0.%d", i)
end
assert_true(blacklist_sync.stamp_ips(batch, false), "stamp_ips accepts capped batch")
local pending = cache_store["_bl_pending"] or ""
assert_true(#pending <= blacklist_sync.PENDING_MAX_BYTES, "pending bytes stay under cap")
assert_true((metric_store.blacklist_pending_immediate_stamp_total or 0) > 0, "overflow IPs immediate-stamped")
assert_true(cache_store["b:10.0.0.20"] == 1, "tail overflow IP stamped at current ver")
assert_true((metric_store.blacklist_pending_overflow_total or 0) == 0, "no drop when _bl_ver set")
assert_true(circuit_errs > 0, "overflow records circuit err")

blacklist_sync.reset_test_hooks()

io.write(string.format("bl_pending_cap_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
