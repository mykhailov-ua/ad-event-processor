-- Role: edge-route-gate env cache at module load.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

ngx = {
    shared = {
        edge_config = {
            get = function()
                return nil
            end,
        },
    },
}

package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-route-gate"] = nil

local edge_route_gate = require "edge-route-gate"

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

edge_route_gate.set_getenv_for_test(function(name)
    if name == "EDGE_EXPOSE_CLICK" then
        return "true"
    end
    return nil
end)
local getenv_calls = 0
edge_route_gate.set_getenv_for_test(function(name)
    getenv_calls = getenv_calls + 1
    if name == "EDGE_EXPOSE_CLICK" then
        return "true"
    end
    return nil
end)
getenv_calls = 0
assert_true(edge_route_gate.click_enabled(), "env true enables click at load")
edge_route_gate.click_enabled()
assert_eq(0, getenv_calls, "click_enabled does not call getenv per request")

edge_route_gate.set_getenv_for_test(function(name)
    if name == "EDGE_EXPOSE_OPENRTB" then
        return "yes"
    end
    return nil
end)
assert_true(edge_route_gate.openrtb_enabled(), "reload picks up openrtb env")

edge_route_gate.set_getenv_for_test(nil)

io.write(string.format("route_gate_env_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
