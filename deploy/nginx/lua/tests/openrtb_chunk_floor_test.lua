-- OpenRTB chunked micro-chunk floor parity with tracker HTTP1_MIN_CHUNKED_DATA_BYTES.
-- Verify: EDGE_MIN_CHUNKED_DATA_BYTES=64 bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local passed = 0
local failed = 0
local exit_status = nil
local exit_thrown = false

ngx = {
    HTTP_BAD_REQUEST = 400,
    HTTP_LENGTH_REQUIRED = 411,
    ERR = 3,
    log = function() end,
    say = function() end,
    header = {},
    ctx = {},
    req = {
        get_headers = function()
            return ngx._test_headers or {}
        end,
        clear_header = function() end,
        socket = function()
            return ngx._test_sock
        end,
    },
    exit = function(code)
        exit_status = code
        exit_thrown = true
        error("ngx.exit", 0)
    end,
    shared = {
        edge_metrics = {
            incr = function(_, key, delta, init)
                ngx._metrics = ngx._metrics or {}
                ngx._metrics[key] = (ngx._metrics[key] or init or 0) + delta
                return ngx._metrics[key]
            end,
        },
        edge_rl = {
            allow = function()
                return true
            end,
            retry_after_sec = function()
                return 60
            end,
        },
        edge_config = {
            get = function()
                return nil
            end,
        },
    },
}

package.loaded["resty.redis"] = {}
package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = require "edge-config"
package.loaded["edge-metrics"] = require "edge-metrics"
package.loaded["edge-fraud-tier"] = {
    sanitize_untrusted_fraud_score_headers = function() end,
    score_for_edge_rl = function()
        return 0
    end,
    tier_from_score = function()
        return "pass"
    end,
}
package.loaded["edge-rl"] = require "edge-rl"
package.loaded["edge-campaign-id"] = {
    resolve_campaign_id = function()
        return "550e8400-e29b-41d4-a716-446655440000"
    end,
    sanitize_untrusted_campaign_id_headers = function() end,
}
package.loaded["edge-parse-dfa"] = {
    MAX_SCAN_BYTES = 8192,
    extract_campaign_id = function()
        return "550e8400-e29b-41d4-a716-446655440000", nil
    end,
}
package.loaded["edge-click-query"] = {}
package.loaded["edge_track_policy"] = nil

local function assert_exit(code, fn, label)
    exit_status = nil
    exit_thrown = false
    local ok, err = pcall(fn)
    if ok or not exit_thrown then
        failed = failed + 1
        io.write("FAIL ", label, ": expected ngx.exit(", code, ") got ", tostring(err), "\n")
        return
    end
    if exit_status ~= code then
        failed = failed + 1
        io.write("FAIL ", label, ": got exit ", tostring(exit_status), "\n")
        return
    end
    passed = passed + 1
end

local lines = { "1", "40" }
local line_idx = 1
ngx._test_sock = {
    settimeout = function() end,
    receive = function(_self, mode)
        if mode == "*l" then
            local line = lines[line_idx]
            line_idx = line_idx + 1
            return line
        end
        return '{"id":"req"}'
    end,
}

ngx._test_headers = {
    ["transfer-encoding"] = "chunked",
}
ngx._metrics = {}

local edge_track_policy = require "edge_track_policy"
assert_exit(400, function()
    edge_track_policy.run_openrtb()
end, "micro_chunk_reject")

if (ngx._metrics.chunked_reject_total or 0) < 1 then
    failed = failed + 1
    io.write("FAIL micro_chunk_metric: chunked_reject_total missing\n")
else
    passed = passed + 1
end

io.write(string.format("openrtb_chunk_floor_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
