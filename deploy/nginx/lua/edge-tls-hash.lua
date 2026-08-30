-- SSL phase hook: MD5 TLS ClientHello record hash into ngx.ctx.tls_hash (fallback JA3 md5).
-- Runtime: nginx worker ssl_client_hello phase; requires edge-tls-fingerprint.compute first.
--
-- Topology: edge-ingress forwards X-TLS-Hash to tracker pool :8181-8184.
--
-- Forbidden: blocking I/O in ssl phase; full handshake reassembly beyond clienthello record.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-tls-hash.lua
-- bash scripts/test/edge/lua_tests.sh
local fp = require "edge-tls-fingerprint"

fp.compute(ngx.ctx)

local ok, ssl_clt = pcall(require, "ngx.ssl.clienthello")
if ok and ssl_clt.get_client_hello_record then
    local der = ssl_clt.get_client_hello_record()
    if der and #der > 0 then
        ngx.ctx.tls_hash = ngx.md5(der)
    end
end

if not ngx.ctx.tls_hash and ngx.ctx.tls_ja3 and ngx.ctx.tls_ja3 ~= "" then
    ngx.ctx.tls_hash = ngx.md5(ngx.ctx.tls_ja3)
end
