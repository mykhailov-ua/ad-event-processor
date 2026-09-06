-- Role: edge_track_policy fail-closed gates (malformed body, stream mode, OPTIONS, TE).
-- Execution context: access phase mocks; apply_parse_error_gate and run_options_track / run_stream.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local metric_store = {}
local rl_store = {}
local exit_status = nil
local exit_thrown = false

ngx = {
    HTTP_BAD_REQUEST = 400,
    HTTP_REQUEST_ENTITY_TOO_LARGE = 413,
    HTTP_TOO_MANY_REQUESTS = 429,
    HTTP_FORBIDDEN = 403,
    WARN = 1,
    INFO = 2,
    ERR = 3,
    time = function()
        return 1000
    end,
    log = function() end,
    say = function() end,
    header = {},
    ctx = {},
    req = {
        get_headers = function()
            return ngx._test_headers or {}
        end,
        clear_header = function() end,
        read_body = function()
            if ngx._test_read_body then
                return ngx._test_read_body()
            end
        end,
    },
    var = {},
    exit = function(code)
        exit_status = code
        exit_thrown = true
        error("ngx.exit", 0)
    end,
    shared = {
        edge_metrics = {
            incr = function(_, key, delta, init)
                metric_store[key] = (metric_store[key] or init or 0) + delta
                return metric_store[key]
            end,
        },
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
                if key == "limit_per_min" then
                    return 1000
                end
                if key == "window_ms" then
                    return 60000
                end
                return nil
            end,
        },
    },
}

package.loaded["resty.redis"] = {}
package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-metrics"] = nil
package.loaded["edge-fraud-tier"] = nil
package.loaded["edge-rl"] = nil
package.loaded["edge-campaign-id"] = nil
package.loaded["edge-parse-dfa"] = nil
package.loaded["edge-click-query"] = nil
package.loaded["edge_track_policy"] = nil

local edge_parse_dfa = require "edge-parse-dfa"
local edge_track_policy = require "edge_track_policy"

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: ", msg, "\n")
    end
end

local function assert_exit(code, fn, msg)
    exit_status = nil
    exit_thrown = false
    local ok = pcall(fn)
    assert_true(not ok and exit_thrown, msg .. " must ngx.exit")
    assert_true(exit_status == code, msg .. " status code")
end

assert_exit(400, function()
    edge_track_policy.apply_parse_error_gate(edge_parse_dfa.ERR_MALFORMED)
end, "malformed_dfa_no_proxy malformed DFA")
assert_true((metric_store.parse_malformed_total or 0) > 0, "malformed_dfa_no_proxy malformed metric")

ngx._test_headers = { ["content-length"] = "10" }
exit_status = nil
exit_thrown = false
metric_store = {}
rl_store = {}
package.loaded["resty.redis"] = {}
package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-metrics"] = nil
package.loaded["edge-fraud-tier"] = nil
package.loaded["edge-rl"] = nil
package.loaded["edge-campaign-id"] = nil
package.loaded["edge_track_policy"] = nil
edge_track_policy = require "edge_track_policy"

assert_exit(400, function()
    edge_track_policy.run_stream()
end, "stream_mode_trusted_header stream mode without trusted header")

ngx._test_read_body = function()
    error "read_body failed"
end
exit_status = nil
exit_thrown = false
metric_store = {}
rl_store = {}
package.loaded["resty.redis"] = {}
package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-metrics"] = nil
package.loaded["edge-fraud-tier"] = nil
package.loaded["edge-rl"] = nil
package.loaded["edge-campaign-id"] = nil
package.loaded["edge_track_policy"] = nil
edge_track_policy = require "edge_track_policy"

assert_exit(400, function()
    edge_track_policy.run_full()
end, "read_body_fail_closed read_body failure")
assert_true((metric_store.body_read_failed_total or 0) > 0, "read_body_fail_closed body read failed metric")

ngx._test_headers = {
    ["content-length"] = "0",
    ["transfer-encoding"] = "gzip",
}
exit_status = nil
exit_thrown = false
metric_store = {}
rl_store = {}
package.loaded["resty.redis"] = {}
package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-metrics"] = nil
package.loaded["edge-fraud-tier"] = nil
package.loaded["edge-rl"] = nil
package.loaded["edge-campaign-id"] = nil
package.loaded["edge_track_policy"] = nil
edge_track_policy = require "edge_track_policy"

assert_exit(411, function()
    edge_track_policy.run_full()
end, "transfer_encoding_reject gzip transfer-encoding on /track")
assert_true((metric_store.chunked_reject_total or 0) > 0, "transfer_encoding_reject gzip TE reject metric")

metric_store = {}
rl_store = {}
package.loaded["resty.redis"] = {}
package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-metrics"] = nil
package.loaded["edge-fraud-tier"] = nil
package.loaded["edge-rl"] = nil
package.loaded["edge-campaign-id"] = nil
package.loaded["edge_track_policy"] = nil
edge_track_policy = require "edge_track_policy"

ngx.var.remote_addr = "203.0.113.88"
ngx.shared.edge_config.get = function(_, key)
    if key == "limit_per_min" then
        return 1
    end
    if key == "window_ms" then
        return 60000
    end
    if key == "rl_pct_suspect" then
        return 100
    end
    return nil
end

edge_track_policy.run_options_track()
assert_true((metric_store.track_policy_pass_total or 0) > 0, "options_track_rate_limit OPTIONS first pass")
exit_status = nil
exit_thrown = false
local ok = pcall(function()
    edge_track_policy.run_options_track()
end)
assert_true(not ok and exit_thrown and exit_status == 429, "options_track_rate_limit OPTIONS second hit rate limited")

io.write(string.format("track_policy_failclosed_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
