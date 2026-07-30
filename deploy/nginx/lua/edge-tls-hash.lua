
local ok, ssl_clt = pcall(require, "ngx.ssl.clienthello")
if not ok then
    return
end

local der = ssl_clt.get_client_hello_record()
if not der or #der == 0 then
    return
end

ngx.ctx.tls_hash = ngx.md5(der)
