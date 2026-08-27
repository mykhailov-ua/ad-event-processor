package.path = arg[1] .. "/?.lua;;"

local args_store = {}

ngx = {
    WARN = 1,
    INFO = 2,
    ERR = 3,
    log = function() end,
    req = {
        get_uri_args = function(_max)
            return args_store, nil
        end,
    },
}

local click_query = require "edge-click-query"

local passed, failed = 0, 0

local function assert_eq(got, want, msg)
    if got == want then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(got), tostring(want)))
    end
end

args_store = { campaign_id = "550e8400-e29b-41d4-a716-446655440000" }
assert_eq(click_query.extract_campaign_id(), "550e8400-e29b-41d4-a716-446655440000", "valid uuid")

args_store = { campaign_id = "550E8400-E29B-41D4-A716-446655440000" }
assert_eq(click_query.extract_campaign_id(), "550e8400-e29b-41d4-a716-446655440000", "lowercases uuid")

args_store = { campaign_id = "not-a-uuid" }
assert_eq(click_query.extract_campaign_id(), nil, "rejects invalid")

args_store = { campaign_id = { "550e8400-e29b-41d4-a716-446655440000", "dup" } }
assert_eq(click_query.extract_campaign_id(), nil, "rejects multi-value")

args_store = {}
assert_eq(click_query.extract_campaign_id(), nil, "missing campaign_id")

io.write(string.format("click_query_test: %d passed, %d failed\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
