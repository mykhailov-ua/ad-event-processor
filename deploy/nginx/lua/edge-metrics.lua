-- Monotonic edge counters in ngx.shared.edge_metrics; Prometheus text on /edge/metrics content handler.
-- Runtime: all workers incr on access phase (access-check, edge_track_policy, edge-ingress, edge-tarpit);
-- render_prometheus() on content handler only (nginx.conf /edge/metrics location).
--
-- Consumers: access-check.lua perimeter/circuit; edge_track_policy.lua body/RL rejects;
-- edge-ingress.lua ingress_protocol; edge-tarpit.lua tarpit_*; blacklist gauges read-only from blacklist_cache.
--
-- Cache invalidation: monotonic incr (no TTL, no delete); keys grow by fixed catalog only.
-- ingress_protocol:{proto}_total uses fixed proto set h1|h2|h3. Fail-open: incr miss logs ERR in ngx.shared impl only.
--
-- ngx.shared edge_metrics (counter keys, number):
-- - perimeter_pass_total, track_policy_pass_total, body_read_total.
-- - circuit_reject_total, blocked_ip_total, blocked_campaign_rl_total, blocked_fraud_tier_total.
-- - parse_oversize_total, body_stream_total, body_peek_total, chunked_reject_total.
-- - ingress_protocol:http/1.1_total, ingress_protocol:h2_total, ingress_protocol:h3_total.
-- - blacklist_stale_total, tarpit_total, tarpit_delay_ms_total.
--
-- ngx.shared blacklist_cache (gauge sources, read-only here):
-- - _bl_sync_ts (number unix s): last successful blacklist stamp -> ad_event_processor_edge_sync_last_success_timestamp.
-- - _bl_count (number): deduped active IP estimate -> ad_event_processor_edge_blacklist_entries.
--
-- ngx.shared edge_config (gauge sources, read-only here):
-- - _asn_cdn_count, _asn_mobile_count (number): stamped ASN keys at last config sync.
-- - dict:free_space() -> ad_event_processor_edge_config_free_bytes.
--
-- State machine: cold zero -> per-request incr on branch taken -> render reads snapshot sums (not reset).
--
-- Constants and limits:
-- - tarpit_delay_ms_total accumulates floor(delay_sec * 1000); negative delay clamped to 0.
-- - Prometheus export prefix ad_event_processor_edge_* (fixed label set; no per-request label keys).
--
-- HTTP status mapping (callers):
-- - 403 blocked IP, fraud tier block; 411 chunked /track; 413 oversize; 429 campaign RL; 503 circuit or blacklist stale.
--
-- Forbidden: dynamic per-request metric label keys beyond fixed ingress_protocol:* set.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-metrics.lua
-- bash scripts/test/edge/lua_tests.sh
local _M = {}

local metrics = ngx.shared.edge_metrics
local blacklist_cache = ngx.shared.blacklist_cache
local edge_config = ngx.shared.edge_config

function _M.record_perimeter_pass()
    metrics:incr("perimeter_pass_total", 1, 0)
end

function _M.record_track_policy_pass()
    metrics:incr("track_policy_pass_total", 1, 0)
end

function _M.record_body_read()
    metrics:incr("body_read_total", 1, 0)
end

function _M.record_circuit_reject()
    metrics:incr("circuit_reject_total", 1, 0)
end

function _M.record_blocked_ip()
    metrics:incr("blocked_ip_total", 1, 0)
end

function _M.record_blocked_campaign_rl()
    metrics:incr("blocked_campaign_rl_total", 1, 0)
end

function _M.record_blocked_fraud_tier()
    metrics:incr("blocked_fraud_tier_total", 1, 0)
end

function _M.record_parse_oversize()
    metrics:incr("parse_oversize_total", 1, 0)
end

function _M.record_body_stream()
    metrics:incr("body_stream_total", 1, 0)
end

function _M.record_body_peek()
    metrics:incr("body_peek_total", 1, 0)
end

function _M.record_chunked_reject()
    metrics:incr("chunked_reject_total", 1, 0)
end

function _M.record_ingress_protocol(proto)
    metrics:incr("ingress_protocol:" .. proto .. "_total", 1, 0)
end

function _M.record_blacklist_stale()
    metrics:incr("blacklist_stale_total", 1, 0)
end

function _M.record_tarpit(delay_sec)
    metrics:incr("tarpit_total", 1, 0)
    local ms = math.floor((delay_sec or 0) * 1000)
    if ms < 0 then
        ms = 0
    end
    metrics:incr("tarpit_delay_ms_total", ms, 0)
end

local function say_metric(name, metric_type, help, value)
    ngx.say("# HELP ", name, " ", help)
    ngx.say("# TYPE ", name, " ", metric_type)
    ngx.say(name, " ", value)
end

function _M.render_prometheus()
    ngx.header["Content-Type"] = "text/plain; version=0.0.4; charset=utf-8"

    local perimeter_pass = metrics:get "perimeter_pass_total" or 0
    local track_policy_pass = metrics:get "track_policy_pass_total" or 0
    local body_read = metrics:get "body_read_total" or 0
    local circuit_reject = metrics:get "circuit_reject_total" or 0
    local blocked_ip = metrics:get "blocked_ip_total" or 0
    local blocked_rl = metrics:get "blocked_campaign_rl_total" or 0
    local blocked_fraud_tier = metrics:get "blocked_fraud_tier_total" or 0
    local parse_oversize = metrics:get "parse_oversize_total" or 0
    local body_stream = metrics:get "body_stream_total" or 0
    local body_peek = metrics:get "body_peek_total" or 0
    local chunked_reject = metrics:get "chunked_reject_total" or 0
    local ingress_h1 = metrics:get "ingress_protocol:http/1.1_total" or 0
    local ingress_h2 = metrics:get "ingress_protocol:h2_total" or 0
    local ingress_h3 = metrics:get "ingress_protocol:h3_total" or 0
    local blacklist_stale = metrics:get "blacklist_stale_total" or 0
    local tarpit_total = metrics:get "tarpit_total" or 0
    local tarpit_delay_ms = metrics:get "tarpit_delay_ms_total" or 0
    local sync_ts = blacklist_cache:get "_bl_sync_ts" or 0
    local bl_count = blacklist_cache:get "_bl_count" or 0
    local asn_cdn_count = edge_config:get "_asn_cdn_count" or 0
    local asn_mobile_count = edge_config:get "_asn_mobile_count" or 0
    local config_free_bytes = edge_config:free_space() or 0

    say_metric(
        "ad_event_processor_edge_perimeter_pass_total",
        "counter",
        "Requests that passed perimeter checks (circuit breaker, IP blacklist).",
        perimeter_pass
    )
    say_metric(
        "ad_event_processor_edge_track_policy_pass_total",
        "counter",
        "Requests that passed body read, parse, and campaign rate-limit checks.",
        track_policy_pass
    )
    say_metric(
        "ad_event_processor_edge_body_read_total",
        "counter",
        "Requests where ngx.req.read_body was invoked at the edge.",
        body_read
    )
    say_metric(
        "ad_event_processor_edge_circuit_reject_total",
        "counter",
        "Requests rejected by edge circuit breaker (503).",
        circuit_reject
    )
    say_metric(
        "ad_event_processor_edge_blocked_ip_total",
        "counter",
        "Requests blocked by IP blacklist at OpenResty edge (403).",
        blocked_ip
    )
    say_metric(
        "ad_event_processor_edge_blocked_campaign_rl_total",
        "counter",
        "Requests blocked by per-campaign edge rate limiter.",
        blocked_rl
    )
    say_metric(
        "ad_event_processor_edge_blocked_fraud_tier_total",
        "counter",
        "Requests blocked by fraud_score tier at edge (403/429).",
        blocked_fraud_tier
    )
    say_metric(
        "ad_event_processor_edge_parse_oversize_total",
        "counter",
        "Requests rejected by edge DFA or Content-Length over body/scan limits (413).",
        parse_oversize
    )
    say_metric(
        "ad_event_processor_edge_body_stream_total",
        "counter",
        "Track policy stream mode: no read_body, body proxied without Lua buffering.",
        body_stream
    )
    say_metric(
        "ad_event_processor_edge_body_peek_total",
        "counter",
        "Track policy peek mode: cosocket read of scan window only.",
        body_peek
    )
    say_metric(
        "ad_event_processor_edge_chunked_reject_total",
        "counter",
        "Requests rejected because chunked encoding is not allowed on edge.",
        chunked_reject
    )
    say_metric(
        "ad_event_processor_edge_ingress_protocol_total",
        "counter",
        "Client ingress protocol at edge (label via separate series below).",
        ingress_h1 + ingress_h2 + ingress_h3
    )
    say_metric(
        "ad_event_processor_edge_ingress_protocol_h1_total",
        "counter",
        "Requests terminated at edge over HTTP/1.1.",
        ingress_h1
    )
    say_metric(
        "ad_event_processor_edge_ingress_protocol_h2_total",
        "counter",
        "Requests terminated at edge over HTTP/2.",
        ingress_h2
    )
    say_metric(
        "ad_event_processor_edge_ingress_protocol_h3_total",
        "counter",
        "Requests terminated at edge over HTTP/3 (QUIC).",
        ingress_h3
    )
    say_metric(
        "ad_event_processor_edge_blacklist_stale_total",
        "counter",
        "Requests rejected because blacklist sync is missing or stale (503).",
        blacklist_stale
    )
    say_metric(
        "ad_event_processor_edge_tarpit_total",
        "counter",
        "Requests delayed by optional edge tarpit (EDGE_TARPIT_ENABLED).",
        tarpit_total
    )
    say_metric(
        "ad_event_processor_edge_tarpit_delay_ms_total",
        "counter",
        "Cumulative tarpit delay milliseconds applied at edge.",
        tarpit_delay_ms
    )
    say_metric(
        "ad_event_processor_edge_sync_last_success_timestamp",
        "gauge",
        "Unix time of last successful blacklist sync from any connected Redis shard.",
        sync_ts
    )
    say_metric(
        "ad_event_processor_edge_blacklist_entries",
        "gauge",
        "Deduped blocked IPs at current blacklist generation (full sync or incremental adds).",
        bl_count
    )
    say_metric(
        "ad_event_processor_edge_asn_cdn_entries",
        "gauge",
        "CDN ASN whitelist stamps written at last edge_config sync.",
        asn_cdn_count
    )
    say_metric(
        "ad_event_processor_edge_asn_mobile_entries",
        "gauge",
        "Mobile ASN whitelist stamps written at last edge_config sync.",
        asn_mobile_count
    )
    say_metric(
        "ad_event_processor_edge_config_free_bytes",
        "gauge",
        "Free bytes remaining in ngx.shared edge_config SHM zone.",
        config_free_bytes
    )
end

return _M
