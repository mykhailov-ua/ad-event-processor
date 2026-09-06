-- Role: edge-rl IP fallback when campaign_id absent.
-- Execution context: ngx.shared.edge_rl; subject key ip:{remote_addr} when campaign_id nil.
-- Invariants proved: nil/empty campaign_id still increments RL bucket; second request denied at limit 1.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local rl_store = {}
local config_store = { limit_per_min = 1, window_ms = 60000 }

ngx = {
    time = function()
        return 1000
    end,
    log = function() end,
    var = {
        remote_addr = "203.0.113.50",
    },
    shared = {
        edge_rl = {
            incr = function(_, key, delta, init)
                local v = rl_store[key]
                if v == nil then
                    v = init or 0
                end
                v = v + delta
                rl_store[key] = v
                return v
            end,
        },
        edge_config = {
            get = function(_, key)
                return config_store[key]
            end,
        },
    },
}

package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-fraud-tier"] = nil
package.loaded["edge-rl"] = nil

local edge_rl = require "edge-rl"

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: ", msg, "\n")
    end
end

local function assert_eq(a, b, msg)
    if a == b then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(a), tostring(b)))
    end
end

assert_eq(
    "550e8400-e29b-41d4-a716-446655440000",
    edge_rl.rl_subject_key "550e8400-e29b-41d4-a716-446655440000",
    "campaign id unchanged"
)
assert_eq("ip:203.0.113.50", edge_rl.rl_subject_key(nil), "nil campaign uses ip fallback")
assert_eq("ip:203.0.113.50", edge_rl.rl_subject_key "", "empty campaign uses ip fallback")

assert_true(edge_rl.allow(nil, 0), "first nil-campaign request allowed")
assert_true(not edge_rl.allow(nil, 0), "second nil-campaign request rate limited")

rl_store = {}
assert_true(edge_rl.allow("550e8400-e29b-41d4-a716-446655440000", 0), "campaign id path still works")
assert_true(not edge_rl.allow("550e8400-e29b-41d4-a716-446655440000", 0), "campaign id second request limited")

io.write(string.format("edge_rl_fallback_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
