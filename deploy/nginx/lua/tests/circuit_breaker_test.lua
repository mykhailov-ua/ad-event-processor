-- Role: edge-circuit breaker bucket math and upstream 5xx err recording.
-- Execution context: ngx.shared.circuit_breaker dict; time mocked via ngx._test_time.
-- Invariants proved: open() requires sample window >= 101 and err rate above threshold; empty upstream_addr skips err.
-- Verify: bash scripts/test/edge/lua_tests.sh all
package.path = arg[1] .. "/?.lua;;"

local circuit_store = {}

local function make_incr_dict(store)
    return {
        get = function(_, key)
            return store[key]
        end,
        incr = function(_, key, delta, init, ttl)
            local val = store[key]
            if val == nil then
                val = init or 0
            end
            val = val + delta
            store[key] = val
            store[key .. ":ttl"] = ttl
            return val
        end,
    }
end

ngx = {
    time = function()
        return ngx._test_time or os.time()
    end,
    shared = {
        circuit_breaker = make_incr_dict(circuit_store),
    },
    var = {},
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

local function reset_store()
    for k in pairs(circuit_store) do
        circuit_store[k] = nil
    end
end

package.loaded["edge-circuit"] = nil
local edge_circuit = require "edge-circuit"

ngx._test_time = 1000
local bucket_curr, bucket_prev = edge_circuit.buckets()
assert_eq(100, bucket_curr, "bucket_curr")
assert_eq(99, bucket_prev, "bucket_prev")

edge_circuit.record_total()
assert_eq(1, circuit_store["100:total"], "record_total increments current bucket")
assert_eq(30, circuit_store["100:total:ttl"], "record_total sets ttl")

edge_circuit.record_err()
assert_eq(1, circuit_store["100:errs"], "record_err increments current bucket")

reset_store()
for _ = 1, 50 do
    edge_circuit.record_total()
end
assert_true(not edge_circuit.open(bucket_curr, bucket_prev), "below sample window stays closed")

reset_store()
for _ = 1, 101 do
    edge_circuit.record_total()
end
for _ = 1, 95 do
    edge_circuit.record_err()
end
assert_true(not edge_circuit.open(bucket_curr, bucket_prev), "95/101 below threshold")

reset_store()
for _ = 1, 101 do
    edge_circuit.record_total()
    edge_circuit.record_err()
end
-- Holdout: 100% error rate in sample window must open circuit (fail-closed to tracker).
assert_true(edge_circuit.open(bucket_curr, bucket_prev), "100% errs opens circuit")

reset_store()
ngx.var.upstream_addr = ""
ngx.var.upstream_status = "502"
edge_circuit.log_upstream_err()
assert_eq(nil, circuit_store["100:errs"], "no upstream_addr skips err")

reset_store()
ngx.var.upstream_addr = "unix:/run/tracker.sock"
ngx.var.upstream_status = "502"
edge_circuit.log_upstream_err()
assert_eq(1, circuit_store["100:errs"], "upstream 5xx records err")

reset_store()
ngx.var.upstream_addr = "unix:/run/tracker.sock"
ngx.var.upstream_status = ""
edge_circuit.log_upstream_err()
assert_eq(1, circuit_store["100:errs"], "empty upstream_status records err")

reset_store()
ngx.var.upstream_addr = "unix:/run/tracker.sock"
ngx.var.upstream_status = "200"
edge_circuit.log_upstream_err()
assert_eq(nil, circuit_store["100:errs"], "upstream 200 skips err")

io.write(string.format("circuit_breaker_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
