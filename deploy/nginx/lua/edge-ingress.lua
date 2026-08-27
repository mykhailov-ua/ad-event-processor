local edge_metrics = require "edge-metrics"

local _M = {}

local function ingress_protocol()
    if ngx.var.http3 == "h3" then
        return "h3"
    end
    if ngx.var.http2 == "h2" then
        return "h2"
    end
    if ngx.var.server_protocol == "HTTP/2.0" then
        return "h2"
    end
    return "http/1.1"
end

function _M.record_and_forward()
    local proto = ingress_protocol()
    edge_metrics.record_ingress_protocol(proto)

    ngx.req.set_header("X-Original-Method", ngx.var.request_method)
    ngx.req.set_header("X-Original-Path", ngx.var.request_uri)

    local tls_hash = ngx.ctx.tls_hash
    if tls_hash and tls_hash ~= "" then
        ngx.req.set_header("X-TLS-Hash", tls_hash)
    elseif ngx.var.ssl_cipher and ngx.var.ssl_cipher ~= "" then
        ngx.req.set_header("X-TLS-Hash", ngx.var.ssl_protocol .. ":" .. ngx.var.ssl_cipher)
    end

    local tls_ja3 = ngx.ctx.tls_ja3
    if tls_ja3 and tls_ja3 ~= "" then
        ngx.req.set_header("X-TLS-JA3", tls_ja3)
    end

    local tls_ja4 = ngx.ctx.tls_ja4
    if tls_ja4 and tls_ja4 ~= "" then
        ngx.req.set_header("X-TLS-JA4", tls_ja4)
    end

    local tcp_fp_cache = ngx.shared.tcp_fp_cache
    local remote = ngx.var.remote_addr

    local mss = ngx.ctx.tcp_mss
    if not mss and tcp_fp_cache then
        mss = tcp_fp_cache:get(remote)
    end
    if mss then
        ngx.req.set_header("X-TCP-MSS", tostring(mss))
    end

    local ttl = ngx.ctx.tcp_ttl
    if not ttl and tcp_fp_cache then
        ttl = tcp_fp_cache:get("t:" .. remote)
    end
    if ttl then
        ngx.req.set_header("X-TCP-TTL", tostring(ttl))
    end

    local win = ngx.ctx.tcp_window
    if not win and tcp_fp_cache then
        win = tcp_fp_cache:get("w:" .. remote)
    end
    if win then
        ngx.req.set_header("X-TCP-WINDOW", tostring(win))
    end
end

return _M
