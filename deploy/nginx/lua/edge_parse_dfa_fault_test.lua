-- Fault harness for edge-parse-dfa.lua contract holdouts (standalone luajit, not nginx worker).
-- Runtime: host luajit via scripts/test/edge/lua_tests.sh; loads edge-parse-dfa from package.path arg[1].
--
-- Contract under test (must match Go parser.mdc budgets):
-- - MAX_BODY_BYTES 1048576, MAX_SCAN_BYTES 8192, MAX_CAMPAIGN_LEN 64.
-- - campaign_id beyond scan window must not extract; ERR_OVERSIZE / ERR_MALFORMED codes.
--
-- Output: edge_parse_dfa_fault: passed=N failed=N faults=N; exit 1 on any holdout failure.
--
-- Verify:
-- luajit deploy/nginx/lua/edge_parse_dfa_fault_test.lua deploy/nginx/lua
-- bash scripts/test/edge/lua_tests.sh edge_parse_dfa_fault
-- go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
package.path = arg[1] .. "/?.lua;;"
local dfa = require "edge-parse-dfa"

local passed, failed, fault_count = 0, 0, 0

local function uuid_bytes()
    return string.char(0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00)
end

local function assert_case(id, name, fn)
    local ok, err = pcall(fn)
    if ok then
        passed = passed + 1
    else
        failed = failed + 1
        fault_count = fault_count + 1
        io.stderr:write(string.format("holdout %s [%s]: %s\n", id, name, tostring(err)))
    end
end

local function expect_nil(cid, err, id, name)
    if cid ~= nil or err ~= nil then
        error(string.format("[%s] %s: expected nil,nil got cid=%s err=%s", id, name, tostring(cid), tostring(err)))
    end
end

local function expect_err(err_code, cid, err, id, name)
    if err ~= err_code then
        error(
            string.format(
                "[%s] %s: expected err=%s got cid=%s err=%s",
                id,
                name,
                err_code,
                tostring(cid),
                tostring(err)
            )
        )
    end
end

local function expect_cid(want, cid, err, id, name)
    if err then
        error(string.format("[%s] %s: unexpected err=%s", id, name, tostring(err)))
    end
    if cid ~= want then
        error(string.format("[%s] %s: want cid=%s got %s", id, name, want, tostring(cid)))
    end
end

assert_case("campaign_id_after_scan_budget", "campaign_id_after_scan_budget", function()
    local junk = string.rep(string.char(0x12, 1, 0x78), 4000)
    local cid_field = string.char(0x0a, 16) .. uuid_bytes()
    local body = junk .. cid_field
    local cid, err = dfa.extract_campaign_id(body, #body)
    if cid ~= nil then
        error "holdout campaign_id_after_scan_budget: campaign_id extracted beyond scan budget"
    end
    if err ~= nil and err ~= dfa.ERR_MALFORMED then
        error("unexpected err " .. tostring(err))
    end
end)

assert_case("varint_bomb", "varint_bomb", function()
    local body = string.char(0x08) .. string.rep(string.char(0x80), 10)
    local cid, err = dfa.extract_campaign_id(body, #body)
    expect_err(dfa.ERR_MALFORMED, cid, err, "E-P02", "varint_bomb")
end)

assert_case("oversize_field_len", "oversize_field_len", function()
    local body = string.char(0x12) .. string.char(0x81, 0x80, 0x04) .. string.rep("a", 100)
    local _, err = dfa.extract_campaign_id(body, #body)
    if err ~= dfa.ERR_OVERSIZE and err ~= dfa.ERR_MALFORMED then
        error("expected oversize or malformed got " .. tostring(err))
    end
end)

assert_case("campaign_id_wrong_field_number", "campaign_id_wrong_field_number", function()
    local body = string.char(0x12, 5) .. "hello" .. string.char(0x12, 16) .. uuid_bytes()
    local cid, err = dfa.extract_campaign_id(body, #body)
    expect_nil(cid, err, "E-P04", "campaign_id_wrong_field_number")
end)

assert_case("binary_uuid_normalized", "binary_uuid_normalized", function()
    local body = string.char(0x0a, 16) .. uuid_bytes()
    local cid, err = dfa.extract_campaign_id(body, #body)
    expect_cid("550e8400-e29b-41d4-a716-446655440000", cid, err, "E-P05", "binary_uuid_normalized")
end)

assert_case("wire_type_3_deprecated", "wire_type_3_deprecated", function()
    local body = string.char(0x1b)
    local cid, err = dfa.extract_campaign_id(body, #body)
    expect_err(dfa.ERR_MALFORMED, cid, err, "E-P08", "wire_type_3_deprecated")
end)

assert_case("truncated_varint", "truncated_varint", function()
    local body = string.char(0x08, 0x80)
    local cid, err = dfa.extract_campaign_id(body, #body)
    expect_err(dfa.ERR_MALFORMED, cid, err, "E-P09", "truncated_varint")
end)

assert_case("json_reordered_keys", "json_reordered_keys", function()
    local json = '{"type":"click","campaign_id":"550e8400-e29b-41d4-a716-446655440000"}'
    local cid, err = dfa.extract_campaign_id(json, #json)
    expect_cid("550e8400-e29b-41d4-a716-446655440000", cid, err, "E-J01", "json_reordered_keys")
end)

assert_case("json_unicode_escape_cid", "json_unicode_escape_cid", function()
    local json = '{"campaign_id":"\\u0035\\u0035\\u0030e8400-e29b-41d4-a716-446655440000"}'
    local cid, _ = dfa.extract_campaign_id(json, #json)
    if cid == "550e8400-e29b-41d4-a716-446655440000" then
        error "holdout: unicode escapes accepted literally without normalization policy"
    end
end)

assert_case("json_null_in_cid", "json_null_in_cid", function()
    local json = '{"campaign_id":"550e8400-e29b-41d4-a716-4466554400\x00"}'
    local cid, _ = dfa.extract_campaign_id(json, #json)
    if cid ~= nil then
        error "holdout: null byte inside campaign_id accepted"
    end
end)

assert_case("json_numeric_campaign_id", "json_numeric_campaign_id", function()
    local json = '{"campaign_id":12345}'
    local cid, err = dfa.extract_campaign_id(json, #json)
    expect_err(dfa.ERR_MALFORMED, cid, err, "E-J06", "json_numeric_campaign_id")
end)

assert_case("json_empty_object", "json_empty_object", function()
    local json = "{}"
    local cid, err = dfa.extract_campaign_id(json, #json)
    expect_nil(cid, err, "E-J08", "json_empty_object")
end)

assert_case("json_duplicate_campaign_id", "json_duplicate_campaign_id", function()
    local json =
        '{"campaign_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","campaign_id":"550e8400-e29b-41d4-a716-446655440000"}'
    local cid, _ = dfa.extract_campaign_id(json, #json)
    if cid ~= "550e8400-e29b-41d4-a716-446655440000" and cid ~= "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" then
        error("unexpected cid " .. tostring(cid))
    end
    if cid == "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" then
        error "holdout json_duplicate_campaign_id: duplicate campaign_id first-wins (expected last-wins for fraud consistency)"
    end
end)

assert_case("check_content_length_oversize", "check_content_length_oversize", function()
    local err = dfa.check_content_length(dfa.MAX_BODY_BYTES + 1)
    expect_err(dfa.ERR_OVERSIZE, nil, err, "E-O02", "check_content_length_oversize")
end)

assert_case("proto_oversize_campaign_field", "proto_oversize_campaign_field", function()
    local body = string.char(0x0a, 100) .. string.rep("a", 100)
    local cid, err = dfa.extract_campaign_id(body, #body)
    expect_err(dfa.ERR_OVERSIZE, cid, err, "E-P06", "proto_oversize_campaign_field")
end)

print(string.format("edge_parse_dfa_fault: passed=%d failed=%d faults=%d", passed, failed, fault_count))
if failed > 0 then
    os.exit(1)
end
