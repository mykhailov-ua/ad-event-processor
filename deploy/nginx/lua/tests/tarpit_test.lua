-- Role: edge-tarpit slow-path delay on oversized headers/body; disabled by default.
-- Execution context: access phase before upstream; EDGE_TARPIT_* env knobs (max headers, body bytes, max sec).
-- Invariants proved: disabled never sleeps; normal requests skip; delay capped at min(EDGE_TARPIT_MAX_SEC, 15s hard max).
-- Verify: bash scripts/test/edge/lua_tests.sh all
package.path = arg[1] .. "/?.lua;;"

local metrics_store = {}
local sleep_calls = {}

ngx = {
    sleep = function(sec)
        sleep_calls[#sleep_calls + 1] = sec
    end,
    req = {
        get_headers = function()
            return ngx._test_headers or {}
        end,
    },
    var = {
        content_length = "0",
    },
}

ngx.shared = {
    edge_metrics = {
        incr = function(_, key, val)
            metrics_store[key] = (metrics_store[key] or 0) + val
        end,
        get = function(_, key)
            return metrics_store[key]
        end,
    },
}

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

local function env_off(_)
    return nil
end

local function env_on(name)
    if name == "EDGE_TARPIT_ENABLED" then
        return "true"
    end
    if name == "EDGE_TARPIT_MAX_HEADERS" then
        return "64"
    end
    if name == "EDGE_TARPIT_BODY_BYTES" then
        return "65536"
    end
    if name == "EDGE_TARPIT_MAX_SEC" then
        return "2"
    end
    return nil
end

package.loaded["edge-tarpit"] = nil
local edge_tarpit = require "edge-tarpit"
edge_tarpit.set_getenv_for_test(env_off)
sleep_calls = {}
metrics_store = {}
ngx._test_headers = {}
for i = 1, 200 do
    ngx._test_headers["X-H-" .. i] = "v"
end
edge_tarpit.maybe_delay()
assert_eq(#sleep_calls, 0, "disabled tarpit must not sleep")
assert_eq(metrics_store.tarpit_total or 0, 0, "disabled tarpit must not record metrics")

package.loaded["edge-tarpit"] = nil
edge_tarpit = require "edge-tarpit"
edge_tarpit.set_getenv_for_test(env_on)
sleep_calls = {}
metrics_store = {}
ngx._test_headers = { Host = "edge", ["User-Agent"] = "test" }
ngx.var.content_length = "1024"
edge_tarpit.maybe_delay()
assert_eq(#sleep_calls, 0, "normal request must not tarpit")

sleep_calls = {}
metrics_store = {}
ngx._test_headers = {}
for i = 1, 100 do
    ngx._test_headers["X-H-" .. i] = "v"
end
ngx.var.content_length = "0"
edge_tarpit.maybe_delay()
assert_true(#sleep_calls == 1, "oversized headers must sleep once")
assert_true(sleep_calls[1] > 0, "delay must be positive")
assert_true((metrics_store.tarpit_total or 0) >= 1, "tarpit_total metric increment")

local delay = edge_tarpit.compute_delay(1000, 0)
assert_true(delay <= 2, "delay capped at EDGE_TARPIT_MAX_SEC")

edge_tarpit.set_getenv_for_test(function(name)
    if name == "EDGE_TARPIT_ENABLED" then
        return "true"
    end
    if name == "EDGE_TARPIT_MAX_SEC" then
        return "99"
    end
    return env_on(name)
end)
delay = edge_tarpit.compute_delay(1000, 0)
assert_true(delay <= 15, "hard max 15s cap")

edge_tarpit.set_getenv_for_test(env_on)
delay = edge_tarpit.compute_delay(1, 200000)
assert_true(delay > 0, "oversized body triggers delay")

io.write(string.format("tarpit_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
