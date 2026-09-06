-- Role: edge-ip canonical() parity with Go net.ParseIP for perimeter blacklist keys.
-- Execution context: access-check, edge-blacklist-sync stamp_ips, edge-rl ip: subject keys.
-- Invariants proved: IPv4-mapped IPv6 unmaps to dotted quad; native IPv4 unchanged; IPv6 inet_ntop canonical.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local edge_ip = require "edge-ip"

local passed, failed = 0, 0

local function assert_eq(got, want, msg)
    if got == want then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s (got %s want %s)\n", msg, tostring(got), tostring(want)))
    end
end

assert_eq(edge_ip.canonical "203.0.113.1", "203.0.113.1", "native ipv4 unchanged")
assert_eq(edge_ip.canonical "::ffff:203.0.113.1", "203.0.113.1", "ipv4-mapped lower")
assert_eq(edge_ip.canonical "::FFFF:203.0.113.1", "203.0.113.1", "ipv4-mapped upper hex")
assert_eq(edge_ip.canonical "2001:db8::1", "2001:db8::1", "native ipv6 compressed")
assert_eq(edge_ip.canonical(nil), nil, "nil passthrough")
assert_eq(edge_ip.canonical "", "", "empty passthrough")

if failed > 0 then
    io.stderr:write(string.format("ip_canonical_test: %d passed, %d failed\n", passed, failed))
    os.exit(1)
end

print(string.format("ip_canonical_test: %d passed", passed))
