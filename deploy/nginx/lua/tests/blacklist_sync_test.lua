package.path = arg[1] .. "/?.lua;;"

package.loaded["resty.redis"] = {}

local blacklist_store = {}
local sentinel_store = {}

local function make_dict(store)
    return {
        get = function(_, key)
            return store[key]
        end,
        set = function(_, key, val)
            store[key] = val
        end,
    }
end

local function make_dict_with_delete(store)
    local dict = make_dict(store)
    dict.delete = function(_, key)
        store[key] = nil
    end
    return dict
end

ngx = {
    WARN = 1,
    INFO = 2,
    ERR = 3,
    log = function() end,
    time = function()
        return os.time()
    end,
    shared = {
        blacklist_cache = make_dict(blacklist_store),
        sentinel_cache = make_dict_with_delete(sentinel_store),
    },
}

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: ", msg, "\n")
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

local blacklist_sync = require("edge-blacklist-sync")

local order = blacklist_sync.shard_try_order(4)
assert_eq(4, #order, "shard_try_order length")
for i = 1, 4 do
    assert_eq(i, order[i], "shard_try_order sequence")
end

blacklist_sync.set_env_for_test(function(name)
    if name == "REDIS_ADDRS" then
        return "bad0:6379,bad1:6379,good2:6379"
    end
    return nil
end)

local attempts = {}
blacklist_sync.set_connect_shard_for_test(function(shard_idx)
    attempts[#attempts + 1] = shard_idx
    if shard_idx < 3 then
        return nil, "down"
    end
    return {
        smembers = function()
            return {}, {}, {}
        end,
        set_keepalive = function() end
    }, nil
end)

local red, err, shard_idx = blacklist_sync.connect_any_shard()
assert_true(red ~= nil, "connect_any_shard succeeds on third shard")
assert_eq(nil, err, "no error on success")
assert_eq(3, shard_idx, "connected shard index")
assert_eq(3, #attempts, "tried shards until success")

blacklist_sync.reset_test_hooks()
blacklist_sync.set_env_for_test(function(name)
    if name == "REDIS_ADDRS" then
        return "bad0:6379,bad1:6379"
    end
    return nil
end)
attempts = {}
blacklist_sync.set_connect_shard_for_test(function(try_idx)
    attempts[#attempts + 1] = try_idx
    return nil, "down"
end)
red, err = blacklist_sync.connect_any_shard()
assert_true(red == nil, "all shards down returns nil client")
assert_true(err ~= nil, "all shards down returns error")
assert_eq(2, #attempts, "attempted every configured shard")

blacklist_sync.reset_test_hooks()
local cache = ngx.shared.blacklist_cache
for k in pairs(blacklist_store) do
    blacklist_store[k] = nil
end

package.loaded["cjson.safe"] = {
    decode = function(body)
        if body == '{"ips":["198.51.100.1","198.51.100.2"]}' then
            return { ips = { "198.51.100.1", "198.51.100.2" } }
        end
        return nil
    end,
}

assert_true(blacklist_sync.stamp_ips({ "203.0.113.5", "203.0.113.6" }), "stamp_ips succeeds")
assert_eq(1, cache:get("_bl_ver"), "stamp_ips bumps version")
assert_eq(1, cache:get("b:203.0.113.5"), "stamp_ips marks first ip")
assert_eq(1, cache:get("b:203.0.113.6"), "stamp_ips marks second ip")

local batch_payload = '{"ips":["198.51.100.1","198.51.100.2"]}'
assert_true(blacklist_sync.apply_quarantine_message(batch_payload), "apply_quarantine_message batch json")
assert_eq(1, cache:get("_bl_ver"), "batch json keeps version on changelog path")
assert_eq(1, cache:get("b:198.51.100.1"), "batch json marks ip")

assert_true(blacklist_sync.apply_quarantine_message("203.0.113.99"), "apply_quarantine_message legacy ip")
assert_eq(1, cache:get("_bl_ver"), "legacy ip keeps version on changelog path")
assert_eq(1, cache:get("b:203.0.113.99"), "legacy ip marks cache")

blacklist_sync.set_env_for_test(function(name)
    if name == "EDGE_BLACKLIST_CHANGELOG_MAX_IPS" then
        return "2"
    end
    return nil
end)
for k in pairs(blacklist_store) do
    if k:sub(1, 3) == "b:" or k == "_bl_pending" then
        blacklist_store[k] = nil
    end
end
cache:set("_bl_ver", 1)
local many = { "10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4" }
assert_true(blacklist_sync.stamp_ips(many, false), "stamp_ips caps changelog batch")
assert_eq(1, cache:get("b:10.0.0.1"), "first ip stamped")
assert_eq(1, cache:get("b:10.0.0.2"), "second ip stamped")
assert_true(blacklist_store["_bl_pending"] ~= nil, "overflow ips queued pending")
blacklist_sync.reset_test_hooks()

io.write(string.format("blacklist_sync_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
