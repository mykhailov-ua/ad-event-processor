-- Role: campaign_id scan-window evasion detection (edge-campaign-id.scan_evasion).
-- Execution context: POST /track when DFA returns nil,nil at MAX_SCAN_BYTES without trusted header.
-- Invariants proved: scan-cap bodies evade; small bodies without id are not evasion.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local edge_parse_dfa = require "edge-parse-dfa"
local edge_campaign_id = require "edge-campaign-id"

local passed, failed = 0, 0
local MAX_SCAN_BYTES = edge_parse_dfa.MAX_SCAN_BYTES

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: " .. msg .. "\n")
    end
end

local function assert_false(cond, msg)
    assert_true(not cond, msg)
end

assert_false(edge_campaign_id.scan_evasion("small", nil, nil, 128), "small body without id is not scan evasion")
assert_true(
    edge_campaign_id.scan_evasion(string.rep("x", MAX_SCAN_BYTES), nil, nil, MAX_SCAN_BYTES),
    "full scan window without id is evasion"
)
assert_true(edge_campaign_id.scan_evasion("", nil, nil, MAX_SCAN_BYTES + 1), "CL beyond scan without id is evasion")

local pad = string.rep(" ", MAX_SCAN_BYTES - 10)
local json = pad .. '{"item":[{"id":"550e8400-e29b-41d4-a716-446655440000"}]}'
local body_id, perr = edge_parse_dfa.extract_campaign_id(json, #json, "openrtb_3")
assert_true(body_id == nil and perr == nil, "openrtb padding fixture returns nil,nil")
assert_true(edge_campaign_id.scan_evasion(json, body_id, perr, #json), "openrtb tail id beyond scan is evasion")

if failed > 0 then
    io.stderr:write(string.format("campaign_id_scan_test: %d passed, %d failed\n", passed, failed))
    os.exit(1)
end

print(string.format("campaign_id_scan_test: %d passed", passed))
