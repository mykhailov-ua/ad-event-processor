-- POST /track, GET /click, POST /openrtb/bid body and framing policy; parity with Go http1TrackEdgePolicy.
-- Runtime: all workers access phase after access-check perimeter (access-check.lua dispatches by uri).
-- Sets ngx.ctx.campaign_id for edge-shard-balancer; does not write ngx.shared.
--
-- Consumers: edge-shard-balancer balance phase; tracker :8181-8184 proxy upstream.
-- Go pair: internal/ingest/httpingress/http1_policy.go http1TrackEdgePolicy (chunked/CL rules on /track).
--
-- Cache invalidation: none (request-scoped ngx.ctx.campaign_id only).
--
-- State machine (uri dispatch from access-check):
-- - /click -> run_click: edge-click-query UUID; optional X-Ad-Event-Processor-Force-Safe without ad click ids.
-- - /openrtb/bid -> run_openrtb: chunked allowed; peek or full body per TE.
-- - else -> run(): EDGE_BODY_MODE full|stream|peek -> run_full|run_stream|run_peek.
--
-- POST /track parity (must match http1TrackEdgePolicy + parser.mdc):
-- - Content-Length required; any Transfer-Encoding (chunked, gzip, etc.) -> 411 chunked_reject metric.
-- - POST /openrtb/bid: chunked allowed; socket peek timeout 500 ms, MAX_SCAN_BYTES window.
--
-- apply_campaign_rl pipeline:
-- - edge-fraud-tier tier block -> 403 fraud block metric + Retry-After.
-- - edge-rl.allow false -> 429 campaign RL metric + Retry-After.
-- - pass -> track_policy_pass metric.
-- - ngx.ctx.campaign_id from edge-campaign-id.resolve_campaign_id (body DFA wins; trusted header fallback).
--
-- Constants and limits:
-- - EDGE_MAX_BODY_BYTES default MAX_SCAN_BYTES 8192 (env EDGE_MAX_BODY_BYTES or /etc/nginx/lua/.edge_max_body_bytes).
-- - INGRESS_SCHEMA TRACKER_INGRESS_SCHEMA default openrtb_3 (env or .edge_ingress_schema file).
-- - EDGE_BODY_MODE default full (env or .edge_body_mode at module load): full|stream|peek.
-- - EDGE_MIN_CHUNKED_DATA_BYTES: OpenRTB chunked data floor (parity HTTP1_MIN_CHUNKED_DATA_BYTES); default 1 disables.
-- - edge-parse-dfa MAX_BODY_BYTES 1048576 for CL oversize; MAX_SCAN_BYTES 8192 for scan/peek.
-- - get_uri_args limit 100 on /click; peek socket timeout 500 ms.
--
-- HTTP: 411 missing CL/chunked /track; 413 oversize; 400 malformed body, campaign_id scan required, body read fail; 429 RL; 403 fraud tier block.
--
-- Forbidden: chunked /track; drift from http1IngressCanonical corpora; claiming edge policy replaces tracker FilterEngine.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge_track_policy.lua
-- bash scripts/test/edge/lua_tests.sh
-- go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
local edge_rl = require "edge-rl"
local edge_metrics = require "edge-metrics"
local edge_parse_dfa = require "edge-parse-dfa"
local edge_fraud_tier = require "edge-fraud-tier"
local edge_click_query = require "edge-click-query"
local edge_campaign_id = require "edge-campaign-id"

local _M = {}

local MAX_SCAN_BYTES = edge_parse_dfa.MAX_SCAN_BYTES

local function bench_file(name)
    local path = "/etc/nginx/lua/.edge_" .. name
    local f = io.open(path, "r")
    if not f then
        return nil
    end
    local v = f:read "*l"
    f:close()
    if v and v ~= "" then
        return v
    end
    return nil
end

local function config_string(name, env_key, default)
    local v = bench_file(name)
    if v then
        return v
    end
    v = os.getenv(env_key)
    if v and v ~= "" then
        return v
    end
    return default
end

local EDGE_MAX_BODY = tonumber(config_string("max_body_bytes", "EDGE_MAX_BODY_BYTES", tostring(MAX_SCAN_BYTES)))
local INGRESS_SCHEMA = config_string("ingress_schema", "TRACKER_INGRESS_SCHEMA", "openrtb_3")
local BODY_MODE = config_string("body_mode", "EDGE_BODY_MODE", "full")
local MIN_CHUNKED_DATA_BYTES = tonumber(config_string("min_chunked_data_bytes", "EDGE_MIN_CHUNKED_DATA_BYTES", "1"))
    or 1

local request_headers

local function refresh_request_headers()
    request_headers = ngx.req.get_headers()
end

local function content_length()
    return tonumber(request_headers["content-length"] or request_headers["Content-Length"])
end

local function header_value(name)
    return request_headers[name] or request_headers[string.lower(name)]
end

-- Parity with http1TrackEdgePolicy: any Transfer-Encoding on POST /track is rejected (chunked, gzip, obfuscated TE).
-- Chunked on POST /openrtb/bid only (run_openrtb).
local function transfer_encoding_present()
    local te = header_value "Transfer-Encoding" or header_value "TE"
    return te ~= nil and te ~= ""
end

local function transfer_encoding_chunked()
    local te = header_value "Transfer-Encoding" or header_value "TE"
    if not te or te == "" then
        return false
    end
    te = string.lower(te)
    if string.find(te, "\x0b", 1, true) or string.find(te, "\t", 1, true) then
        return false
    end
    return string.find(te, "chunked", 1, true) ~= nil
end

local function fraud_score_from_headers()
    edge_fraud_tier.sanitize_untrusted_fraud_score_headers()
    return edge_fraud_tier.score_for_edge_rl()
end

local function reject_parse_malformed()
    edge_metrics.record_parse_malformed()
    ngx.status = ngx.HTTP_BAD_REQUEST
    ngx.say "malformed request body"
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

local function reject_body_unavailable()
    edge_metrics.record_body_read_failed()
    ngx.status = ngx.HTTP_BAD_REQUEST
    ngx.say "request body unavailable"
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

local function reject_oversize()
    edge_metrics.record_parse_oversize()
    ngx.exit(ngx.HTTP_REQUEST_ENTITY_TOO_LARGE)
end

local function handle_parse_error(perr)
    if perr == edge_parse_dfa.ERR_OVERSIZE then
        reject_oversize()
    end
    if perr == edge_parse_dfa.ERR_MALFORMED then
        reject_parse_malformed()
    end
end

local function reject_chunked()
    edge_metrics.record_chunked_reject()
    ngx.status = ngx.HTTP_LENGTH_REQUIRED
    ngx.say "Content-Length required"
    ngx.exit(411)
end

local function reject_invalid_content_length()
    edge_metrics.record_chunked_reject()
    ngx.status = ngx.HTTP_BAD_REQUEST
    ngx.say "invalid content-length"
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

local function reject_cl_te_conflict()
    edge_metrics.record_chunked_reject()
    ngx.status = ngx.HTTP_BAD_REQUEST
    ngx.say "conflicting content-length and transfer-encoding"
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

local function reject_micro_chunk()
    edge_metrics.record_chunked_reject()
    ngx.status = ngx.HTTP_BAD_REQUEST
    ngx.say "chunked body too small"
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

-- Parity with tracker ParseHTTP1ChunkedBody MinChunkedDataBytes (HTTP1_MIN_CHUNKED_DATA_BYTES / EDGE_MIN_CHUNKED_DATA_BYTES).
local function enforce_openrtb_chunk_floor(sock)
    local floor = MIN_CHUNKED_DATA_BYTES
    if not floor or floor <= 1 then
        return
    end
    sock:settimeout(500)
    local line, err = sock:receive "*l"
    if not line then
        ngx.log(ngx.ERR, "edge openrtb chunk size: ", err or "unknown")
        reject_body_unavailable()
    end
    if string.find(line, ";", 1, true) then
        reject_parse_malformed()
    end
    line = string.gsub(line, "[%s\t]", "")
    local size = tonumber(line, 16)
    if not size then
        reject_parse_malformed()
    end
    if size > 0 and size < floor then
        reject_micro_chunk()
    end
end

local function check_edge_limits(cl)
    if cl and cl < 0 then
        reject_invalid_content_length()
    end
    if edge_parse_dfa.check_content_length(cl) == edge_parse_dfa.ERR_OVERSIZE then
        reject_oversize()
    end
    if edge_parse_dfa.check_content_length(cl) == edge_parse_dfa.ERR_MALFORMED then
        reject_invalid_content_length()
    end
    if cl and cl > EDGE_MAX_BODY then
        reject_oversize()
    end
end

-- POST /track: Content-Length mandatory; any Transfer-Encoding rejected (TestChaos_CrossHop_NginxGnet differential_count=0).
local function require_content_length()
    if transfer_encoding_present() then
        reject_chunked()
    end
    local cl = content_length()
    if not cl then
        reject_chunked()
    end
    return cl
end

local function reject_rate_limited(fraud_score)
    edge_metrics.record_blocked_campaign_rl()
    ngx.header["Retry-After"] = tostring(edge_rl.retry_after_sec(fraud_score))
    ngx.status = ngx.HTTP_TOO_MANY_REQUESTS
    ngx.say "rate limit exceeded"
    ngx.exit(ngx.HTTP_TOO_MANY_REQUESTS)
end

local function reject_fraud_block(fraud_score)
    edge_metrics.record_blocked_fraud_tier()
    ngx.header["Retry-After"] = tostring(edge_rl.retry_after_sec(fraud_score))
    ngx.status = ngx.HTTP_FORBIDDEN
    ngx.say "fraud score block"
    ngx.exit(ngx.HTTP_FORBIDDEN)
end

-- Edge RL pipeline before proxy: fraud tier block -> 403; edge_rl deny -> 429 Retry-After from edge-config.
-- Sets ngx.ctx.campaign_id upstream of edge-shard-balancer; no ngx.shared writes here.
local function apply_campaign_rl(campaign_id, fraud_score)
    local tier = edge_fraud_tier.tier_from_score(fraud_score or 0)
    if tier == "block" then
        reject_fraud_block(fraud_score)
    end
    if not edge_rl.allow(campaign_id, fraud_score) then
        reject_rate_limited(fraud_score)
    end
end

local function finish_policy_without_body(fraud_score, stream_metric)
    local campaign_id = edge_campaign_id.resolve_campaign_id(nil)
    if campaign_id and campaign_id ~= "" then
        ngx.ctx.campaign_id = campaign_id
    end
    apply_campaign_rl(campaign_id, fraud_score)
    if stream_metric then
        edge_metrics.record_body_stream()
    end
    edge_metrics.record_track_policy_pass()
end

local function reject_campaign_id_scan_required()
    edge_metrics.record_campaign_id_scan_reject()
    ngx.status = ngx.HTTP_BAD_REQUEST
    ngx.say "campaign_id required"
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

-- campaign_id_scan_evasion: nil DFA id at scan/CL cap without trusted header must not fall back to IP-only edge_rl.
-- malformed_dfa_no_proxy: ERR_MALFORMED from DFA must not proxy to tracker.
local function resolve_campaign_id_for_rl(body, body_id, perr, cl)
    handle_parse_error(perr)
    local campaign_id = edge_campaign_id.resolve_campaign_id(body_id)
    if campaign_id and campaign_id ~= "" then
        return campaign_id
    end
    if edge_campaign_id.scan_evasion(body, body_id, perr, cl) then
        reject_campaign_id_scan_required()
    end
    return campaign_id
end

local function apply_track_campaign_policy(body, body_id, perr, cl, fraud_score)
    local campaign_id = resolve_campaign_id_for_rl(body, body_id, perr, cl)
    if campaign_id and campaign_id ~= "" then
        ngx.ctx.campaign_id = campaign_id
    end
    apply_campaign_rl(campaign_id, fraud_score)
end

local function read_bounded_body(cl)
    check_edge_limits(cl)
    edge_metrics.record_body_read()

    local read_ok, read_err = pcall(ngx.req.read_body)
    if not read_ok then
        ngx.log(ngx.ERR, "failed to read body: ", read_err)
        reject_body_unavailable()
    end

    local body = ngx.req.get_body_data()
    if not body then
        local filename = ngx.req.get_body_file()
        if filename then
            local fh = io.open(filename, "rb")
            if fh then
                body = fh:read(EDGE_MAX_BODY + 1)
                fh:close()
                if body and #body > EDGE_MAX_BODY then
                    reject_oversize()
                end
            end
        end
    elseif #body > EDGE_MAX_BODY then
        reject_oversize()
    end
    if not body and cl and cl > 0 then
        reject_body_unavailable()
    end
    return body, cl
end

function _M.run_full()
    refresh_request_headers()
    local cl = require_content_length()
    local body, _ = read_bounded_body(cl)
    local fraud_score = fraud_score_from_headers()
    local body_id, perr = edge_parse_dfa.extract_campaign_id(body, cl, INGRESS_SCHEMA)
    apply_track_campaign_policy(body, body_id, perr, cl, fraud_score)
    edge_metrics.record_track_policy_pass()
end

function _M.run_stream()
    refresh_request_headers()
    local cl = require_content_length()
    check_edge_limits(cl)
    local fraud_score = fraud_score_from_headers()
    edge_campaign_id.sanitize_untrusted_campaign_id_headers()
    local campaign_id = edge_campaign_id.resolve_campaign_id(nil)
    if not campaign_id or campaign_id == "" then
        reject_campaign_id_scan_required()
    end
    ngx.ctx.campaign_id = campaign_id
    apply_campaign_rl(campaign_id, fraud_score)
    edge_metrics.record_body_stream()
    edge_metrics.record_track_policy_pass()
end

function _M.run_peek()
    refresh_request_headers()
    local cl = require_content_length()
    check_edge_limits(cl)

    local sock, sock_err = ngx.req.socket()
    if not sock then
        ngx.log(ngx.ERR, "edge peek: socket unavailable: ", sock_err)
        reject_body_unavailable()
    end

    sock:settimeout(500)
    local chunk = sock:receive(MAX_SCAN_BYTES)
    edge_metrics.record_body_peek()

    local fraud_score = fraud_score_from_headers()
    if chunk and #chunk > 0 then
        local body_id, perr = edge_parse_dfa.extract_campaign_id(chunk, cl, INGRESS_SCHEMA)
        apply_track_campaign_policy(chunk, body_id, perr, cl, fraud_score)
        edge_metrics.record_track_policy_pass()
    else
        finish_policy_without_body(fraud_score, false)
    end
end

function _M.run_click()
    refresh_request_headers()
    local args, err = ngx.req.get_uri_args(100)
    if not args then
        ngx.log(ngx.ERR, "edge click query parse failed: ", err or "unknown")
        args = {}
    end
    local fraud_score = fraud_score_from_headers()
    local campaign_id = edge_click_query.extract_campaign_id()
    if campaign_id and campaign_id ~= "" then
        ngx.ctx.campaign_id = campaign_id
    end

    if not args.gclid and not args.fbclid and not args.ttclid and not args.yclid and not args.w then
        ngx.req.set_header("X-Ad-Event-Processor-Force-Safe", "1")
    end

    apply_campaign_rl(campaign_id, fraud_score)
    edge_metrics.record_track_policy_pass()
end

function _M.run_openrtb()
    refresh_request_headers()
    local fraud_score = fraud_score_from_headers()
    if transfer_encoding_chunked() then
        local cl = content_length()
        if cl then
            reject_cl_te_conflict()
        end
        local sock, sock_err = ngx.req.socket()
        if not sock then
            ngx.log(ngx.ERR, "edge openrtb peek: socket unavailable: ", sock_err)
            reject_body_unavailable()
        end
        enforce_openrtb_chunk_floor(sock)
        sock:settimeout(500)
        local chunk = sock:receive(MAX_SCAN_BYTES)
        edge_metrics.record_body_peek()
        if chunk and #chunk > 0 then
            local body_id, perr = edge_parse_dfa.extract_campaign_id(chunk, cl, INGRESS_SCHEMA)
            apply_track_campaign_policy(chunk, body_id, perr, cl, fraud_score)
            edge_metrics.record_track_policy_pass()
        else
            finish_policy_without_body(fraud_score, false)
        end
        return
    end

    local cl = require_content_length()
    local body, _ = read_bounded_body(cl)
    local body_id, perr = edge_parse_dfa.extract_campaign_id(body, cl, INGRESS_SCHEMA)
    apply_track_campaign_policy(body, body_id, perr, cl, fraud_score)
    edge_metrics.record_track_policy_pass()
end

function _M.apply_parse_error_gate(perr)
    handle_parse_error(perr)
end

function _M.run_options_track()
    local fraud_score = fraud_score_from_headers()
    apply_campaign_rl(nil, fraud_score)
    edge_metrics.record_track_policy_pass()
end

function _M.run()
    local mode = BODY_MODE
    if mode == "stream" then
        _M.run_stream()
    elseif mode == "peek" then
        _M.run_peek()
    else
        _M.run_full()
    end
end

return _M
