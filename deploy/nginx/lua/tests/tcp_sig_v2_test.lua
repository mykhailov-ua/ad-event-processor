-- Role: edge-tcp-sig-v2 trace validation parity with pkg/tcpsynopt NormalizeTokens.
-- Execution context: luajit standalone via lua_tests.sh; no ngx runtime required.
-- Invariants proved: lower-case normalize, empty/oversize/control-char reject, token cap 64.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local tcp_sig_v2 = require "edge-tcp-sig-v2"

local passed, failed = 0, 0

local function assert_eq(a, b, msg)
    if a == b then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(a), tostring(b)))
    end
end

local function assert_nil(v, msg)
    if v == nil then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s)\n", msg, tostring(v)))
    end
end

assert_eq(tcp_sig_v2.validate_trace "nop,nop,sackok,mss:1460", "nop,nop,sackok,mss:1460", "linux trace")
assert_eq(tcp_sig_v2.validate_trace "  NOP , MSS:1440 ", "nop,mss:1440", "trim and lower-case")
assert_nil(tcp_sig_v2.validate_trace "", "empty trace")
assert_nil(tcp_sig_v2.validate_trace(string.rep("a", 513)), "trace too long")
assert_nil(tcp_sig_v2.validate_trace "ok\nbad", "control char rejected")
assert_nil(tcp_sig_v2.validate_trace(string.rep("x", 65)), "token too long")

if failed > 0 then
    os.exit(1)
end

print(string.format("tcp_sig_v2_test: %d passed", passed))
