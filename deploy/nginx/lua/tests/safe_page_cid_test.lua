-- Role: edge-safe-page campaign_id normalization.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

package.loaded["edge-safe-page"] = nil
local edge_safe_page = require "edge-safe-page"

local passed, failed = 0, 0

local function assert_eq(a, b, msg)
    if a == b then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(a), tostring(b)))
    end
end

assert_eq("", edge_safe_page.normalize_campaign_id(nil), "nil -> empty")
assert_eq("", edge_safe_page.normalize_campaign_id "<script>alert(1)</script>", "reject injection string")
assert_eq(
    "550e8400-e29b-41d4-a716-446655440000",
    edge_safe_page.normalize_campaign_id "550E8400-E29B-41D4-A716-446655440000",
    "lowercase uuid"
)

io.write(string.format("safe_page_cid_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
