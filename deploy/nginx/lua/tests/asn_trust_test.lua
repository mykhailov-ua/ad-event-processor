-- Role: edge-asn trusted header contract.
-- Execution context: access-check calls sanitize_untrusted_asn_headers then trusted_client_asn.
-- Invariants proved: client X-Client-ASN ignored; X-Edge-Trusted-ASN drives whitelist only.
-- Verify: bash scripts/test/edge/lua_tests.sh unit
package.path = arg[1] .. "/?.lua;;"

local config_store = {
    _asn_ver = 2,
    ["asn_cdn:15169"] = 2,
}

local cleared_headers = {}
local ngx_vars = {}

ngx = {
    null = {},
    var = setmetatable({}, {
        __index = function(_, key)
            return ngx_vars[key]
        end,
    }),
    req = {
        clear_header = function(name)
            cleared_headers[#cleared_headers + 1] = name
            if name == "X-Client-ASN" then
                ngx_vars.http_x_client_asn = nil
            end
        end,
    },
    shared = {
        edge_config = {
            get = function(_, key)
                return config_store[key]
            end,
        },
    },
}

package.loaded["edge-blacklist-sync"] = {}
package.loaded["edge-circuit"] = { record_err = function() end }
package.loaded["edge-config"] = nil
package.loaded["edge-asn"] = nil

local edge_config = require "edge-config"
local edge_asn = require "edge-asn"

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

local function reset_request()
    cleared_headers = {}
    ngx_vars = {}
end

reset_request()
ngx_vars.http_x_client_asn = "15169"
edge_asn.sanitize_untrusted_asn_headers()
assert_eq("X-Client-ASN", cleared_headers[1], "sanitize clears client asn header")
assert_eq(nil, edge_asn.trusted_client_asn(), "spoofed X-Client-ASN not read after sanitize")
assert_true(not edge_config.asn_whitelisted(edge_asn.trusted_client_asn()), "no bypass without trusted header")

reset_request()
ngx_vars.http_x_edge_trusted_asn = "15169"
edge_asn.sanitize_untrusted_asn_headers()
assert_eq("15169", edge_asn.trusted_client_asn(), "trusted header value used")
assert_true(edge_asn.is_whitelisted(edge_asn.trusted_client_asn()), "trusted asn whitelisted")

local function perimeter_blocked(client_ip, trusted_asn)
    if edge_asn.is_whitelisted(trusted_asn) then
        return false
    end
    local ver = 9
    local ip_ver = client_ip == "203.0.113.5" and ver or nil
    return ip_ver and ip_ver == ver
end

reset_request()
ngx_vars.http_x_client_asn = "15169"
edge_asn.sanitize_untrusted_asn_headers()
assert_true(
    perimeter_blocked("203.0.113.5", edge_asn.trusted_client_asn()),
    "blocked ip stays blocked with spoof header"
)

reset_request()
ngx_vars.http_x_edge_trusted_asn = "15169"
edge_asn.sanitize_untrusted_asn_headers()
assert_true(not perimeter_blocked("203.0.113.5", edge_asn.trusted_client_asn()), "trusted asn skips blacklist gate")

edge_asn.set_getenv_for_test(function(name)
    if name == "EDGE_TRUSTED_ASN_HEADER" then
        return "X-Client-ASN"
    end
    return nil
end)
reset_request()
ngx_vars.http_x_client_asn = "15169"
cleared_headers = {}
edge_asn.sanitize_untrusted_asn_headers()
assert_eq(0, #cleared_headers, "cdn hop mode keeps X-Client-ASN when configured as trusted")
assert_eq("15169", edge_asn.trusted_client_asn(), "cdn hop reads configured trusted header")
edge_asn.set_getenv_for_test(nil)

io.write(string.format("asn_trust_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
