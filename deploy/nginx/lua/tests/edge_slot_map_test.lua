package.path = arg[1] .. "/?.lua;;"

local slot_map_store = {}
local set_log = {}

local slot_map_dict = {
    get = function(_, key)
        return slot_map_store[key]
    end,
    set = function(_, key, val)
        set_log[#set_log + 1] = { key, val }
        slot_map_store[key] = val
    end,
}

ngx = {
    WARN = 1,
    INFO = 2,
    ERR = 3,
    log = function() end,
    shared = {
        slot_map = slot_map_dict,
    },
}

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: " .. msg .. "\n")
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
    for k in pairs(slot_map_store) do
        slot_map_store[k] = nil
    end
    for i = #set_log, 1, -1 do
        set_log[i] = nil
    end
end

local function make_slots(shard)
    local slots = {}
    for i = 1, 1024 do
        slots[i] = shard
    end
    return slots
end

package.loaded["edge-net"] = {
    http_get_json = function(_)
        return {
            version = 42,
            routing_epoch = 99,
            slots = make_slots(7),
        }
    end,
}

package.loaded["edge-slot-map"] = nil
local slot_map = require "edge-slot-map"

local campaign_id = "550e8400-e29b-41d4-a716-446655440000"

reset_store()
slot_map_store["version"] = 10
for i = 0, 1023 do
    slot_map_store["s:" .. i] = 3
end
assert_true(slot_map.get_shard(campaign_id) ~= nil, "get_shard returns shard when version > 0")

reset_store()
slot_map_store["version"] = 0
for i = 0, 1023 do
    slot_map_store["s:" .. i] = 3
end
assert_true(slot_map.get_shard(campaign_id) == nil, "get_shard nil when version is 0")

reset_store()
for i = 0, 1023 do
    slot_map_store["s:" .. i] = 3
end
assert_true(slot_map.get_shard(campaign_id) == nil, "get_shard nil when version missing")

reset_store()
slot_map_store["version"] = 10
for i = 0, 1023 do
    slot_map_store["s:" .. i] = 3
end
slot_map_store["routing_epoch"] = 5

slot_map.sync()

assert_eq(slot_map_store["version"], 42, "sync bumps version")
assert_eq(slot_map_store["routing_epoch"], 99, "sync updates routing_epoch")
for i = 0, 1023 do
    assert_eq(slot_map_store["s:" .. i], 7, "sync slot " .. i)
end

local version_idx = nil
local last_s_idx = nil
for i, entry in ipairs(set_log) do
    if entry[1] == "version" then
        version_idx = i
    end
    if string.sub(entry[1], 1, 2) == "s:" then
        last_s_idx = i
    end
end
assert_true(version_idx ~= nil, "sync sets version")
assert_true(last_s_idx ~= nil, "sync sets s:* keys")
assert_true(version_idx > last_s_idx, "version set after all s:* keys")

reset_store()
slot_map_store["version"] = 10
for i = 0, 1023 do
    slot_map_store["s:" .. i] = 3
end

local mid_version = nil
local mid_shard = nil
local orig_set = slot_map_dict.set
slot_map_dict.set = function(self, key, val)
    if mid_version == nil and string.sub(key, 1, 2) == "s:" then
        mid_version = slot_map_store["version"]
        mid_shard = slot_map.get_shard(campaign_id)
    end
    return orig_set(self, key, val)
end

slot_map.sync()
slot_map_dict.set = orig_set

assert_eq(mid_version, 10, "mid-sync get_shard sees old version")
assert_eq(mid_shard, 3, "mid-sync get_shard reads prior slot map")
assert_eq(slot_map_store["version"], 42, "post-sync version matches doc")

io.write(string.format("edge_slot_map_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
