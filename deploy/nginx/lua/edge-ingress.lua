-- Forward ingress protocol, TLS, and TCP fingerprint signals to tracker upstream request headers.
-- Runtime: all workers access phase (access-check.lua after tarpit, before edge_track_policy).
-- No ngx.shared writes; reads tcp_fp_cache populated by edge-tcp-fp-sync worker 0 timer.
--
-- Consumers: tracker pool :8181-8184 via proxy; edge-metrics.record_ingress_protocol on each request.
-- TLS ctx from ssl phases: edge-tls-hash.lua, edge-tls-fingerprint.lua set ngx.ctx.tls_* before access.
--
-- Cache invalidation (read path): tcp_fp_cache TTL 3600 s per key from edge-tcp-fp-sync sync;
-- miss fail-open (header omitted). ngx.ctx tcp_* overrides SHM when set in ssl phase.
--
-- ngx.shared tcp_fp_cache keys (types, remote_addr = client IP string):
-- - {ip} (number 0..255): MSS -> X-TCP-MSS.
-- - t:{ip} (number 0..255): TTL -> X-TCP-TTL.
-- - w:{ip} (number 0..65535): window -> X-TCP-WINDOW.
-- - h:{ip} (string 8 hex): tcp_hash -> X-TCP-SIG.
--
-- ngx.ctx inputs (optional, override SHM):
-- - tls_hash, tls_ja3, tls_ja4, tls_alpn; tcp_mss, tcp_ttl, tcp_window, tcp_sig.
--
-- State machine: record ingress proto -> set X-Original-Method/Path -> TLS headers -> TCP fp -> timing headers.
--
-- Upstream headers when data present:
-- - X-Original-Method, X-Original-Path.
-- - X-TLS-Hash (ctx or ssl_protocol:ssl_cipher), X-TLS-JA3, X-TLS-JA4, X-TLS-ALPN.
-- - X-TCP-MSS, X-TCP-TTL, X-TCP-WINDOW, X-TCP-SIG.
-- - X-TTFB-APP-MS from connection_time (1..65535 ms); X-RTT-SYN-MS from tcpinfo_rtt us.
--
-- Constants and limits:
-- - ingress_protocol labels: h3 (http3), h2 (http2 or HTTP/2.0), else http/1.1.
-- - TTFB/RTT headers only when computed ms in 1..65535.
--
-- CDN/L4 LB: TCP SYN signals absent without edge-xdp + tcp_fp_sync; tracker may skip OS fingerprint checks.
--
-- Forbidden: blocking Redis cosocket in access phase; per-request tcp_fp_sync.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-ingress.lua
-- bash scripts/test/edge/lua_tests.sh
-- go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
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

-- Ingress header forward: read-only tcp_fp_cache (worker 0 edge-tcp-fp-sync); ngx.ctx overrides SHM.
-- Miss on SHM is fail-open (header omitted). XDP/edge-bpf-sync supplies Redis staging; not per-request Redis.
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

    local tls_alpn = ngx.ctx.tls_alpn
    if tls_alpn and tls_alpn ~= "" then
        ngx.req.set_header("X-TLS-ALPN", tls_alpn)
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

    local sig = ngx.ctx.tcp_sig
    if not sig and tcp_fp_cache then
        sig = tcp_fp_cache:get("h:" .. remote)
    end
    if sig and sig ~= "" then
        ngx.req.set_header("X-TCP-SIG", sig)
    end

    -- Connection timing for tracker cold-path rtt_split_tunnel (CH rtt_syn_ms, ttfb_app_ms).
    -- Emit only when computed ms in 1..65535; absent when nginx vars unset or out of range.
    local conn_time = tonumber(ngx.var.connection_time)
    if conn_time and conn_time > 0 then
        local ttfb_ms = math.floor(conn_time * 1000 + 0.5)
        if ttfb_ms > 0 and ttfb_ms <= 65535 then
            ngx.req.set_header("X-TTFB-APP-MS", tostring(ttfb_ms))
        end
    end

    local rtt_us = tonumber(ngx.var.tcpinfo_rtt)
    if rtt_us and rtt_us > 0 then
        local rtt_ms = math.floor((rtt_us + 500) / 1000)
        if rtt_ms > 0 and rtt_ms <= 65535 then
            ngx.req.set_header("X-RTT-SYN-MS", tostring(rtt_ms))
        end
    end
end

return _M
