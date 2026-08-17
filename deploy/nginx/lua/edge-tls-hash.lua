
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
