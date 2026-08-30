-- L7 perimeter access phase: circuit breaker, generational IP blacklist, ASN bypass, tarpit, ingress, route dispatch.
-- Runtime: every nginx worker; single-threaded Lua VM per worker; reads ngx.shared only (no Redis cosocket here).
--
-- Pipeline: circuit_breaker -> perimeter_blacklist -> tarpit -> edge-ingress -> edge_track_policy -> proxy.
-- Blacklist SHM populated by edge-blacklist-sync (worker 0); optional XDP L4 is separate (known host/CIDR only).
--
-- ngx.shared circuit_breaker (10 s wall-clock buckets, key TTL 30 s on incr); logic in edge-circuit.lua:
-- - {bucket}:total incremented once per request via edge_circuit.record_total().
-- - {bucket}:errs incremented by edge-circuit writers on:
--   * edge-blacklist-sync connect_any_shard exhaustion and sync connect/smembers failures;
--   * edge-config sync connect/hmget failures;
--   * perimeter_blacklist missing or stale _bl_sync_ts (503, separate from blacklist metric);
--   * edge-circuit-log log_by_lua upstream 5xx or empty upstream_status when upstream_addr set.
-- - open when (errs_curr+errs_prev)/(total_curr+total_prev) > 0.95 after 100 combined samples.
-- - Below sample window: fail-open (breaker treated closed).
-- - Open: 503 fail-closed.
--
-- ngx.shared blacklist_cache generational check (perimeter_blacklist):
-- - Missing _bl_ver or _bl_sync_ts: 503 fail-closed (no successful sync yet).
-- - ngx.time() - _bl_sync_ts > BL_STALE_SEC (EDGE_BL_STALE_SEC default 30): 503 fail-closed.
-- - ip_ver = get("b:" .. client_ip); 403 fail-closed only when ip_ver and ip_ver == _bl_ver.
-- - ip_ver present but ip_ver ~= _bl_ver: fail-open (stale stamp from prior generation after unblock).
-- - No b:{ip} key: fail-open.
--
-- ASN fail-open whitelist (edge-asn.lua, not tracker FilterEngine bypass):
-- - X-Client-ASN whitelisted via edge_config asn_cdn:* / asn_mobile:* skips perimeter_blacklist only.
-- - Missing or non-whitelisted ASN: full generational blacklist check applies.
--
-- HTTP outcomes: 403 blocked IP; 503 circuit open or blacklist missing/stale; pass records perimeter_pass metric.
--
-- Forbidden: synchronous Redis in access phase; per-request blacklist refresh; treating ASN whitelist as fraud pass on tracker.
--
-- Verify:
-- luac -p deploy/nginx/lua/access-check.lua
-- bash scripts/test/edge/lua_tests.sh
-- go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
local edge_metrics = require "edge-metrics"
local edge_circuit = require "edge-circuit"
local edge_track_policy = require "edge_track_policy"
local edge_asn = require "edge-asn"
local edge_ingress = require "edge-ingress"
local edge_tarpit = require "edge-tarpit"

local blacklist_cache = ngx.shared.blacklist_cache

local BL_STALE_SEC = tonumber(os.getenv "EDGE_BL_STALE_SEC" or "") or 30

local function client_asn()
    return ngx.var.http_x_client_asn
end

local function perimeter_blacklist(client_ip)
    if edge_asn.is_whitelisted(client_asn()) then
        return
    end

    local ver = blacklist_cache:get "_bl_ver"
    local sync_ts = blacklist_cache:get "_bl_sync_ts"

    if not ver or not sync_ts then
        edge_circuit.record_err()
        edge_metrics.record_blacklist_stale()
        ngx.log(ngx.ERR, "edge blacklist: no successful sync yet")
        ngx.exit(ngx.HTTP_SERVICE_UNAVAILABLE)
    end

    if ngx.time() - sync_ts > BL_STALE_SEC then
        edge_circuit.record_err()
        edge_metrics.record_blacklist_stale()
        ngx.log(ngx.ERR, "edge blacklist: sync stale > ", BL_STALE_SEC, "s")
        ngx.exit(ngx.HTTP_SERVICE_UNAVAILABLE)
    end

    local ip_ver = blacklist_cache:get("b:" .. client_ip)
    if ip_ver and ip_ver == ver then
        edge_metrics.record_blocked_ip()
        ngx.exit(ngx.HTTP_FORBIDDEN)
    end
end

local function perimeter_gate(client_ip, bucket_curr, bucket_prev)
    edge_circuit.record_total()
    if edge_circuit.open(bucket_curr, bucket_prev) then
        edge_metrics.record_circuit_reject()
        ngx.log(ngx.ERR, "Edge Circuit Breaker OPEN")
        ngx.exit(ngx.HTTP_SERVICE_UNAVAILABLE)
    end

    perimeter_blacklist(client_ip)
    edge_metrics.record_perimeter_pass()
end

local bucket_curr, bucket_prev = edge_circuit.buckets()
local client_ip = ngx.var.remote_addr

perimeter_gate(client_ip, bucket_curr, bucket_prev)
edge_tarpit.maybe_delay()
edge_ingress.record_and_forward()

local edge_route_gate = require "edge-route-gate"
local uri = ngx.var.uri
if ngx.req.get_method() == "OPTIONS" and uri == "/track" then
    return
end
if uri == "/click" then
    edge_route_gate.require_click()
    edge_track_policy.run_click()
elseif uri == "/openrtb/bid" then
    edge_route_gate.require_openrtb()
    edge_track_policy.run_openrtb()
else
    edge_track_policy.run()
end
