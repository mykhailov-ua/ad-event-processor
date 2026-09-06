-- Role: edge-fraud-tier trusted header contract.
-- Execution context: edge_track_policy fraud_score_from_headers clears X-Fraud-Score then reads trusted header.
-- Invariants proved: client X-Fraud-Score ignored; trusted header drives tier block and RL scaling.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local cleared_headers = {}
local ngx_vars = {}

ngx = {
    var = setmetatable({}, {
        __index = function(_, key)
            return ngx_vars[key]
        end,
    }),
    req = {
        clear_header = function(name)
            cleared_headers[#cleared_headers + 1] = name
            if name == "X-Fraud-Score" then
                ngx_vars.http_x_fraud_score = nil
            end
        end,
    },
}

package.loaded["edge-fraud-tier"] = nil
local edge_fraud_tier = require "edge-fraud-tier"

local passed, failed = 0, 0

local function assert_eq(a, b, msg)
    if a == b then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(a), tostring(b)))
    end
end

local function reset_request()
    cleared_headers = {}
    ngx_vars = {}
end

local function fraud_score_from_headers()
    edge_fraud_tier.sanitize_untrusted_fraud_score_headers()
    return edge_fraud_tier.score_for_edge_rl()
end

reset_request()
ngx_vars.http_x_fraud_score = "0"
edge_fraud_tier.sanitize_untrusted_fraud_score_headers()
assert_eq("X-Fraud-Score", cleared_headers[1], "sanitize clears client fraud score header")
local tier, _ = edge_fraud_tier.tier_from_score(edge_fraud_tier.score_for_edge_rl())
assert_eq("pass", tier, "spoofed client zero ignored without trusted header")

reset_request()
ngx_vars.http_x_fraud_score = "0"
ngx_vars.http_x_edge_trusted_fraud_score = "95"
local score = fraud_score_from_headers()
local tier_trusted, _ = edge_fraud_tier.tier_from_score(score)
assert_eq("block", tier_trusted, "trusted high score maps to block tier despite client zero")
assert_eq(95, score, "trusted score value used")

reset_request()
local score_missing = fraud_score_from_headers()
assert_eq(31, score_missing, "missing trusted score defaults to suspect tier input")
local tier_missing, _ = edge_fraud_tier.tier_from_score(score_missing)
assert_eq("suspect", tier_missing, "missing trusted score is suspect tier RL")

edge_fraud_tier.set_getenv_for_test(function(name)
    if name == "EDGE_TRUSTED_FRAUD_SCORE_HEADER" then
        return "X-Fraud-Score"
    end
    return nil
end)
reset_request()
ngx_vars.http_x_fraud_score = "95"
cleared_headers = {}
edge_fraud_tier.sanitize_untrusted_fraud_score_headers()
assert_eq(0, #cleared_headers, "inner hop mode keeps X-Fraud-Score when configured trusted")
assert_eq("block", edge_fraud_tier.tier_from_score(edge_fraud_tier.score_for_edge_rl()))
edge_fraud_tier.set_getenv_for_test(nil)

io.write(string.format("fraud_score_trust_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
