-- Role: edge-campaign-id trusted header contract.
-- Execution context: edge_track_policy resolve_campaign_id clears X-Campaign-Id then reads trusted header.
-- Invariants proved: client header ignored; body id wins; trusted header normalized to lowercase UUID.
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
            if name == "X-Campaign-Id" then
                ngx_vars.http_x_campaign_id = nil
            end
        end,
    },
}

package.loaded["edge-campaign-id"] = nil
local edge_campaign_id = require "edge-campaign-id"

local valid_uuid = "550e8400-e29b-41d4-a716-446655440000"
local valid_upper = "550E8400-E29B-41D4-A716-446655440000"

local passed, failed = 0, 0

local function assert_eq(a, b, msg)
    if a == b then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(a), tostring(b)))
    end
end

local function assert_nil(a, msg)
    if a == nil then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want nil)\n", msg, tostring(a)))
    end
end

local function reset_request()
    cleared_headers = {}
    ngx_vars = {}
end

reset_request()
ngx_vars.http_x_campaign_id = valid_uuid
edge_campaign_id.sanitize_untrusted_campaign_id_headers()
assert_eq("X-Campaign-Id", cleared_headers[1], "sanitize clears client campaign id header")
assert_nil(edge_campaign_id.trusted_campaign_id(), "spoofed X-Campaign-Id not read after sanitize")

reset_request()
ngx_vars.http_x_edge_trusted_campaign_id = valid_upper
edge_campaign_id.sanitize_untrusted_campaign_id_headers()
assert_eq(valid_uuid, edge_campaign_id.trusted_campaign_id(), "trusted header normalized to lowercase uuid")

reset_request()
ngx_vars.http_x_edge_trusted_campaign_id = "not-a-uuid; DROP TABLE"
assert_nil(edge_campaign_id.trusted_campaign_id(), "invalid trusted header rejected")

reset_request()
ngx_vars.http_x_campaign_id = valid_uuid
ngx_vars.http_x_edge_trusted_campaign_id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
local body_id = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
assert_eq(body_id, edge_campaign_id.resolve_campaign_id(body_id), "body id wins over trusted header")
assert_eq("X-Campaign-Id", cleared_headers[1], "resolve clears client header")

reset_request()
ngx_vars.http_x_campaign_id = valid_uuid
assert_nil(edge_campaign_id.resolve_campaign_id(nil), "stream fallback ignores client header without trusted id")

reset_request()
ngx_vars.http_x_edge_trusted_campaign_id = valid_uuid
assert_eq(valid_uuid, edge_campaign_id.resolve_campaign_id(nil), "stream fallback uses trusted header")

edge_campaign_id.set_getenv_for_test(function(name)
    if name == "EDGE_TRUSTED_CAMPAIGN_ID_HEADER" then
        return "X-Campaign-Id"
    end
    return nil
end)
reset_request()
ngx_vars.http_x_campaign_id = valid_upper
cleared_headers = {}
assert_eq(valid_uuid, edge_campaign_id.resolve_campaign_id(nil), "inner hop mode reads configured trusted header")
assert_eq(0, #cleared_headers, "inner hop mode keeps X-Campaign-Id when configured trusted")
edge_campaign_id.set_getenv_for_test(nil)

io.write(string.format("campaign_id_trust_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
