package.path = arg[1] .. "/?.lua;;"

package.loaded["resty.redis"] = {}

local config_store = {}
local circuit_store = {}

local function make_dict(store)
    return {
        get = function(_, key)
            return store[key]
        end,
        set = function(_, key, val)
            store[key] = val
            return true, nil, false
        end,
        delete = function(_, key)
            store[key] = nil
        end,
        free_space = function()
            return 1048576
        end,
    }
end

local function make_incr_dict(store)
    return {
        get = function(_, key)
            return store[key]
        end,
        incr = function(_, key, delta, init)
            local val = store[key]
            if val == nil then
                val = init or 0
            end
            val = val + delta
            store[key] = val
            return val
        end,
    }
end

ngx = {
    WARN = 1,
    null = {},
    log = function() end,
    shared = {
        edge_config = make_dict(config_store),
        circuit_breaker = make_incr_dict(circuit_store),
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

local function assert_nil(a, msg)
    if a == nil then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want nil)\n", msg, tostring(a)))
    end
end

local function reset_config()
    for k in pairs(config_store) do
        config_store[k] = nil
    end
end

package.loaded["edge-circuit"] = {
    record_err = function() end,
}

local blacklist_sync = require "edge-blacklist-sync"
local edge_config = require "edge-config"

assert_true(not edge_config.redis_value_ok(nil), "redis_value_ok rejects nil")
assert_true(not edge_config.redis_value_ok(ngx.null), "redis_value_ok rejects ngx.null")
assert_true(not edge_config.redis_value_ok(""), "redis_value_ok rejects empty string")
assert_true(edge_config.redis_value_ok("0"), "redis_value_ok accepts zero string")
assert_true(edge_config.redis_value_ok(0), "redis_value_ok accepts zero number")

assert_true(edge_config.truthy_flag("true"), "truthy_flag true")
assert_true(edge_config.truthy_flag("1"), "truthy_flag 1")
assert_true(not edge_config.truthy_flag("false"), "truthy_flag false")
assert_true(not edge_config.truthy_flag("0"), "truthy_flag 0")
assert_true(not edge_config.truthy_flag(ngx.null), "truthy_flag ngx.null")

local function mock_hmget(vals)
    blacklist_sync.set_connect_shard_for_test(function()
        return {
            hmget = function()
                return vals, nil
            end,
            set_keepalive = function() end,
        },
            nil
    end)
end

reset_config()
config_store.limit_per_min = 42
mock_hmget {
    ngx.null,
    ngx.null,
    ngx.null,
    ngx.null,
    ngx.null,
    ngx.null,
    ngx.null,
    ngx.null,
    "15169",
    ngx.null,
    "true",
    ngx.null,
}
edge_config.sync()
assert_eq(42, config_store.limit_per_min, "null numerics retain prior limit")
assert_eq(1, config_store._asn_ver, "asn generation bumped on sync")
assert_eq(1, config_store["asn_cdn:15169"], "cdn asn stamped with new generation")
assert_true(edge_config.asn_whitelisted("15169"), "asn_whitelisted matches current generation")
assert_eq(1, config_store.edge_expose_click, "truthy expose click stored as 1")
assert_nil(config_store.edge_expose_openrtb, "null expose openrtb deleted")

reset_config()
config_store["asn_cdn:15169"] = 1
config_store._asn_ver = 1
mock_hmget {
    "200",
    "30000",
    "50",
    "10",
    "0",
    "30",
    "60",
    "120",
    "",
    "",
    "0",
    "false",
}
edge_config.sync()
assert_eq(2, config_store._asn_ver, "empty asn csv bumps generation")
assert_eq(1, config_store["asn_cdn:15169"], "stale asn stamp not deleted")
assert_true(not edge_config.asn_whitelisted("15169"), "stale asn stamp no longer whitelisted")
assert_eq(200, config_store.limit_per_min, "numeric limit updated")
assert_eq(0, config_store.edge_expose_click, "false expose click stored as 0")
assert_eq(0, config_store.edge_expose_openrtb, "false expose openrtb stored as 0")

reset_config()
mock_hmget {
    "100",
    "60000",
    "50",
    "10",
    "0",
    "30",
    "60",
    "120",
    "15169, 20940",
    "31000",
    "yes",
    "1",
}
edge_config.sync()
assert_eq(1, config_store._asn_ver, "first full sync generation")
assert_eq(2, config_store._asn_cdn_count, "cdn asn count tracked")
assert_eq(1, config_store._asn_mobile_count, "mobile asn count tracked")
assert_eq(1, config_store["asn_cdn:15169"], "first cdn stamp")
assert_eq(1, config_store["asn_mobile:31000"], "mobile stamp")
assert_true(edge_config.asn_whitelisted("20940"), "trimmed cdn asn whitelisted")
assert_true(edge_config.asn_whitelisted("31000"), "mobile asn whitelisted")

mock_hmget {
    "100",
    "60000",
    "50",
    "10",
    "0",
    "30",
    "60",
    "120",
    "20940",
    ngx.null,
    ngx.null,
    ngx.null,
}
edge_config.sync()
assert_eq(2, config_store._asn_ver, "second sync bumps generation after stamps")
assert_eq(1, config_store._asn_cdn_count, "cdn count after partial restamp")
assert_eq(0, config_store._asn_mobile_count, "mobile count zero when redis null clears active stamps")
assert_eq(2, config_store["asn_cdn:20940"], "restamped cdn asn at new generation")
assert_eq(1, config_store["asn_cdn:15169"], "removed cdn asn left stale")
assert_true(not edge_config.asn_whitelisted("15169"), "removed asn not whitelisted")
assert_true(edge_config.asn_whitelisted("20940"), "restamped asn whitelisted")
assert_nil(config_store.edge_expose_click, "null expose flag deleted for env fallback")
assert_nil(config_store.edge_expose_openrtb, "null expose openrtb deleted")

blacklist_sync.reset_test_hooks()

io.write(string.format("edge_config_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
