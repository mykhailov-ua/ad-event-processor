
package.path = arg[1] .. "/?.lua;;"

package.loaded["resty.redis"] = {}

ngx = {
    log = {
        WARN = function() end,
        INFO = function() end,
        ERR = function() end,
    },
    time = function()
        return os.time()
    end,
    shared = {
        blacklist_cache = {
            _store = {},
            get = function(self, key)
                return self._store[key]
            end,
            set = function(self, key, val)
                self._store[key] = val
            end,
        },
        sentinel_cache = {
            _store = {},
            get = function(self, key)
                return self._store[key]
            end,
            set = function(self, key, val)
                self._store[key] = val
            end,
            delete = function(self, key)
                self._store[key] = nil
            end,
        },
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
    return { smembers = function()
        return {}, {}, {}
    end, set_keepalive = function() end }, nil
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
blacklist_sync.set_connect_shard_for_test(function(shard_idx)
    attempts[#attempts + 1] = shard_idx
    return nil, "down"
end)
red, err = blacklist_sync.connect_any_shard()
assert_true(red == nil, "all shards down returns nil client")
assert_true(err ~= nil, "all shards down returns error")
assert_eq(2, #attempts, "attempted every configured shard")

io.write(string.format("blacklist_sync_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
