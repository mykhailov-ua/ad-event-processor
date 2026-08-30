-- Role: edge-uuid normalize/normalize_to_bytes and slot_map get_shard reject invalid campaign_id.
-- Execution context: click and shard routing; must match Go UUID parse fail-closed (nil, no throw on hot path).
-- Invariants proved: invalid hex never normalizes; normalize_to_bytes invalid does not throw; get_shard returns nil on bad uuid.
-- Verify: bash scripts/test/edge/lua_tests.sh all
package.path = arg[1] .. "/?.lua;;"

local edge_uuid = require "edge-uuid"

local passed, failed = 0, 0
local VALID = "550e8400-e29b-41d4-a716-446655440000"

local function assert_eq(got, want, msg)
    if got == want then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(got), tostring(want)))
    end
end

local function assert_nil(got, msg)
    assert_eq(got, nil, msg)
end

local function assert_ok(fn, msg)
    local ok, err = pcall(fn)
    if ok then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s panicked: %s\n", msg, tostring(err)))
    end
end

assert_eq(edge_uuid.normalize(VALID), VALID, "accepts lowercase uuid")
assert_eq(edge_uuid.normalize "550E8400-E29B-41D4-A716-446655440000", VALID, "lowercases uuid")
assert_nil(edge_uuid.normalize "not-a-uuid", "rejects garbage")
assert_nil(edge_uuid.normalize "gggggggg-gggg-gggg-gggg-gggggggggggg", "rejects non-hex")
assert_nil(edge_uuid.normalize "550e8400-e29b-41d4-a716-44665544000g", "rejects bad last nibble")
assert_nil(edge_uuid.normalize(VALID .. "x"), "rejects too long")
assert_nil(edge_uuid.normalize "550e8400-e29b-41d4-a716-44665544000", "rejects too short")

local bytes = edge_uuid.normalize_to_bytes(VALID)
assert_eq(bytes and #bytes, 16, "produces 16 raw bytes")

assert_ok(function()
    edge_uuid.normalize_to_bytes "gggggggg-gggg-gggg-gggg-gggggggggggg"
end, "invalid hex does not throw")

package.loaded["edge-slot-map"] = nil
ngx = {
    shared = {
        slot_map = {
            _data = { version = 1 },
            get = function(self, key)
                return self._data[key]
            end,
            set = function(self, key, val)
                self._data[key] = val
            end,
        },
    },
}
local slot_map = require "edge-slot-map"
assert_nil(slot_map.get_shard "gggggggg-gggg-gggg-gggg-gggggggggggg", "slot_map rejects invalid uuid")
assert_ok(function()
    slot_map.get_shard "gggggggg-gggg-gggg-gggg-gggggggggggg"
end, "slot_map get_shard does not throw on invalid uuid")

io.write(string.format("uuid_test: %d passed, %d failed\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
