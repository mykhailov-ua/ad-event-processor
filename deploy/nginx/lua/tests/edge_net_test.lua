-- Role: edge-net redis_connect UDS vs TCP host/port wiring for state Redis shards.
-- Execution context: OpenResty worker; luajit standalone via lua_tests.sh (mock resty.redis).
-- Invariants proved: unix_socket maps to unix:/path host; TCP passes host and numeric port separately.
-- Verify: bash scripts/test/edge/lua_tests.sh all
package.path = arg[1] .. "/?.lua;;"

local edge_net = require "edge-net"

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: ", msg, "\n")
    end
end

local captured = {}

local function mock_redis()
    return {
        set_timeout = function() end,
        connect = function(_, host, port_or_opts, opts)
            captured.host = host
            captured.port_or_opts = port_or_opts
            captured.opts = opts
            return true
        end,
    }
end

local red = mock_redis()
local ok = edge_net.redis_connect(red, { unix_socket = "/run/ad-event-processor/redis/redis-0.sock" })
assert_true(ok == true, "unix redis_connect returns true")
assert_true(captured.host == "unix:/run/ad-event-processor/redis/redis-0.sock", "unix host includes path")
assert_true(captured.port_or_opts ~= nil and type(captured.port_or_opts) == "table", "unix connect passes opts table")

captured = {}
red = mock_redis()
ok = edge_net.redis_connect(red, { host = "127.0.0.1", port = 6479 })
assert_true(ok == true, "tcp redis_connect returns true")
assert_true(captured.host == "127.0.0.1", "tcp host")
assert_true(captured.port_or_opts == 6479, "tcp port number")

if failed > 0 then
    os.exit(1)
end

print(string.format("edge_net_test: %d passed", passed))
