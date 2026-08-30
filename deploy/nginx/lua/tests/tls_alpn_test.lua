-- Role: TLS ClientHello ALPN extension parser for edge TLS fingerprinting.
-- Execution context: edge-tls-fingerprint on handshake metadata; pure Lua bit unpack.
-- Invariants proved: parse_alpn_list returns comma-separated protocols; empty/short ext returns "" without error.
-- Verify: bash scripts/test/edge/lua_tests.sh all
package.path = arg[1] .. "/?.lua;;"

local tls_fp = require "edge-tls-fingerprint"

local passed, failed = 0, 0

local function assert_eq(got, want, msg)
    if got == want then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: ", msg, " got=", tostring(got), " want=", tostring(want), "\n")
    end
end

local h2 = string.char(2) .. "h2"
local http11 = string.char(8) .. "http/1.1"
local ext = string.char(0, #h2 + #http11) .. h2 .. http11
assert_eq(tls_fp.parse_alpn_list(ext), "h2,http/1.1", "dual alpn")

assert_eq(tls_fp.parse_alpn_list "", "", "empty ext")
assert_eq(tls_fp.parse_alpn_list(string.char(0)), "", "short ext")

if failed > 0 then
    os.exit(1)
end

print(string.format("tls_alpn_test: %d passed", passed))
