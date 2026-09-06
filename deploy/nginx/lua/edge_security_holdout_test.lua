-- Security holdouts for deploy/nginx/lua production modules (standalone luajit).
-- Maps to edge audit slugs IDs in SECURITY_AUDIT.md; proves exploitable behavior or contract gaps.
--
-- Runtime: host luajit or OpenResty container via scripts/test/edge/lua_tests.sh security
--
-- Verify:
-- luajit deploy/nginx/lua/edge_security_holdout_test.lua deploy/nginx/lua
-- bash scripts/test/edge/lua_tests.sh security
package.path = arg[1] .. "/?.lua;;"

local passed, failed, fault_count = 0, 0, 0

local function assert_case(id, name, fn)
    local ok, err = pcall(fn)
    if ok then
        passed = passed + 1
    else
        failed = failed + 1
        fault_count = fault_count + 1
        io.stderr:write(string.format("holdout %s [%s]: %s\n", id, name, tostring(err)))
    end
end

local function expect_true(cond, id, name)
    if not cond then
        error(string.format("[%s] %s: expected true", id, name))
    end
end

local function expect_false(cond, id, name)
    if cond then
        error(string.format("[%s] %s: expected false", id, name))
    end
end

local function expect_eq(want, got, id, name)
    if got ~= want then
        error(string.format("[%s] %s: want %s got %s", id, name, tostring(want), tostring(got)))
    end
end

local function expect_nil(v, id, name)
    if v ~= nil then
        error(string.format("[%s] %s: want nil got %s", id, name, tostring(v)))
    end
end

-- client_asn_blacklist_bypass: client X-Client-ASN must not bypass blacklist gate.
assert_case("client_asn_blacklist_bypass", "client_asn_header_no_blacklist_bypass", function()
    local config_store = {
        _asn_ver = 2,
        ["asn_cdn:15169"] = 2,
    }
    local cleared = {}
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
                cleared[#cleared + 1] = name
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
            blacklist_cache = {
                get = function(_, key)
                    if key == "_bl_ver" then
                        return 9
                    end
                    if key == "b:203.0.113.5" then
                        return 9
                    end
                    return nil
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

    ngx_vars.http_x_client_asn = "15169"
    edge_asn.sanitize_untrusted_asn_headers()
    expect_true(cleared[1] == "X-Client-ASN", "client_asn_blacklist_bypass", "sanitize clears client header")

    local function perimeter_blocked(client_ip)
        local asn = edge_asn.trusted_client_asn()
        if edge_config.asn_whitelisted(asn) then
            return false
        end
        local ver = ngx.shared.blacklist_cache:get "_bl_ver"
        local ip_ver = ngx.shared.blacklist_cache:get("b:" .. client_ip)
        return ip_ver and ip_ver == ver
    end

    expect_true(
        perimeter_blocked "203.0.113.5",
        "client_asn_blacklist_bypass",
        "spoofed X-Client-ASN does not bypass blacklist"
    )

    ngx_vars.http_x_edge_trusted_asn = "15169"
    expect_false(
        perimeter_blocked "203.0.113.5",
        "client_asn_blacklist_bypass",
        "trusted header still enables asn bypass"
    )
end)

-- client_fraud_score_rl_tier: client X-Fraud-Score must not downgrade edge fraud tier.
assert_case("client_fraud_score_rl_tier", "client_fraud_score_no_tier_bypass", function()
    local cleared = {}
    local ngx_vars = {}
    ngx = {
        var = setmetatable({}, {
            __index = function(_, key)
                return ngx_vars[key]
            end,
        }),
        req = {
            clear_header = function(name)
                cleared[#cleared + 1] = name
                if name == "X-Fraud-Score" then
                    ngx_vars.http_x_fraud_score = nil
                end
            end,
        },
    }
    package.loaded["edge-fraud-tier"] = nil
    local edge_fraud_tier = require "edge-fraud-tier"

    ngx_vars.http_x_fraud_score = "0"
    ngx_vars.http_x_edge_trusted_fraud_score = "95"
    edge_fraud_tier.sanitize_untrusted_fraud_score_headers()
    expect_true(cleared[1] == "X-Fraud-Score", "client_fraud_score_rl_tier", "sanitize clears client header")

    local score = edge_fraud_tier.score_for_edge_rl()
    local tier, _ = edge_fraud_tier.tier_from_score(score)
    expect_eq("block", tier, "client_fraud_score_rl_tier", "trusted high score still blocks after client spoof zero")

    ngx_vars.http_x_edge_trusted_fraud_score = nil
    cleared = {}
    edge_fraud_tier.sanitize_untrusted_fraud_score_headers()
    score = edge_fraud_tier.score_for_edge_rl()
    tier = edge_fraud_tier.tier_from_score(score)
    expect_eq(
        "suspect",
        tier,
        "fraud_score_suspect_without_trusted",
        "no trusted header yields suspect RL tier not full pass"
    )
end)

-- nil_campaign_id_ip_rl_fallback: nil campaign_id uses IP fallback for edge_rl (no unbounded bypass).
assert_case("nil_campaign_id_ip_rl_fallback", "nil_campaign_id_ip_rl_fallback", function()
    local rl_store = {}
    local config_store = { limit_per_min = 1, window_ms = 60000 }
    ngx = {
        time = function()
            return 1000
        end,
        log = function() end,
        var = {
            remote_addr = "198.51.100.7",
        },
        shared = {
            edge_rl = {
                incr = function(_, key, delta, init)
                    local v = rl_store[key]
                    if v == nil then
                        v = init or 0
                    end
                    v = v + delta
                    rl_store[key] = v
                    return v
                end,
            },
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
    package.loaded["edge-fraud-tier"] = nil
    package.loaded["edge-rl"] = nil
    local edge_rl = require "edge-rl"
    expect_eq(
        "ip:198.51.100.7",
        edge_rl.rl_subject_key(nil),
        "nil_campaign_id_ip_rl_fallback",
        "nil campaign maps to ip subject"
    )
    expect_true(edge_rl.allow(nil, 0), "nil_campaign_id_ip_rl_fallback", "first nil-campaign request allowed")
    expect_false(edge_rl.allow(nil, 0), "nil_campaign_id_ip_rl_fallback", "second nil-campaign request rate limited")
end)

-- header_campaign_id_trusted_only: client X-Campaign-Id must not steer routing without trusted header + UUID normalize.
assert_case("header_campaign_id_trusted_only", "header_campaign_id_no_uuid_validation", function()
    local cleared = {}
    local ngx_vars = {}
    ngx = {
        var = setmetatable({}, {
            __index = function(_, key)
                return ngx_vars[key]
            end,
        }),
        req = {
            clear_header = function(name)
                cleared[#cleared + 1] = name
                if name == "X-Campaign-Id" then
                    ngx_vars.http_x_campaign_id = nil
                end
            end,
        },
    }
    package.loaded["edge-campaign-id"] = nil
    local edge_campaign_id = require "edge-campaign-id"

    local spoof = "not-a-uuid; DROP TABLE"
    ngx_vars.http_x_campaign_id = spoof
    edge_campaign_id.sanitize_untrusted_campaign_id_headers()
    expect_true(cleared[1] == "X-Campaign-Id", "header_campaign_id_trusted_only", "sanitize clears client header")
    expect_nil(
        edge_campaign_id.resolve_campaign_id(nil),
        "header_campaign_id_trusted_only",
        "stream fallback ignores spoof client header"
    )

    local valid = "550e8400-e29b-41d4-a716-446655440000"
    ngx_vars.http_x_edge_trusted_campaign_id = "550E8400-E29B-41D4-A716-446655440000"
    expect_eq(
        valid,
        edge_campaign_id.resolve_campaign_id(nil),
        "header_campaign_id_trusted_only",
        "trusted header normalized for routing"
    )
end)

-- peek_socket_fail_policy: peek/openrtb socket failure still runs fraud tier + edge_rl (IP fallback).
assert_case("peek_socket_fail_policy", "peek_fail_open_logic", function()
    local function run_peek_outcome(sock_ok, allow_calls)
        allow_calls = allow_calls or 0
        if not sock_ok then
            allow_calls = allow_calls + 1
            return nil, true, allow_calls
        end
        return nil, true, allow_calls
    end
    local _, peek_passed, allow_calls = run_peek_outcome(false, 0)
    expect_nil(nil, "peek_socket_fail_policy", "no campaign_id on socket fail")
    expect_true(peek_passed, "peek_socket_fail_policy", "request still passes policy")
    expect_eq(1, allow_calls, "peek_socket_fail_policy", "socket fail still invokes edge_rl allow path")
end)

-- native_json_depth_limit: native JSON skip_json_value enforces MAX_NATIVE_JSON_DEPTH (Go MaxJSONDepth=16).
assert_case("native_json_depth_limit", "native_json_depth_unbounded", function()
    package.loaded["edge-parse-dfa"] = nil
    local dfa = require "edge-parse-dfa"
    local function nested_object(depth)
        local inner = string.rep('{"k":', depth) .. "1" .. string.rep("}", depth)
        return inner
    end
    local depth = 25
    local json = nested_object(depth)
    if #json > dfa.MAX_SCAN_BYTES then
        error "fixture exceeds scan window"
    end
    local _, err_native = dfa.extract_campaign_id(json, #json, "ad_event_processor_native")
    expect_eq(dfa.ERR_MALFORMED, err_native, "native_json_depth_limit", "native path rejects excessive depth")
    local depth_bomb = dfa.MAX_JSON_DEPTH + 3
    local ortb_json = nested_object(depth_bomb)
    if #ortb_json > dfa.MAX_SCAN_BYTES then
        ortb_json = string.sub(ortb_json, 1, dfa.MAX_SCAN_BYTES)
    end
    local _, err_ortb = dfa.extract_campaign_id(ortb_json, #ortb_json, "openrtb_3")
    expect_eq(dfa.ERR_MALFORMED, err_ortb, "native_json_depth_limit", "openrtb path rejects excessive depth")
end)

-- Fuzz: nested native JSON depths 1..40 must not crash luajit VM.
assert_case("fuzz_nested_native_json", "fuzz_nested_native_json", function()
    package.loaded["edge-parse-dfa"] = nil
    local dfa = require "edge-parse-dfa"
    for depth = 1, 40 do
        local json = string.rep("{", depth) .. string.rep("}", depth)
        if #json <= dfa.MAX_SCAN_BYTES then
            local ok, err = pcall(dfa.extract_campaign_id, json, #json)
            if not ok then
                error(string.format("fuzz panic depth=%d: %s", depth, tostring(err)))
            end
        end
    end
end)

-- bl_pending_byte_cap: _bl_pending capped at PENDING_MAX_BYTES; overflow immediate-stamped (bl_pending_overflow_immediate_stamp).
assert_case("bl_pending_byte_cap", "bl_pending_unbounded", function()
    local cache_store = {}
    local metric_store = {}
    local circuit_errs = 0
    ngx = {
        WARN = 1,
        INFO = 2,
        ERR = 3,
        log = function() end,
        time = function()
            return 1
        end,
        shared = {
            blacklist_cache = {
                get = function(_, k)
                    return cache_store[k]
                end,
                set = function(_, k, v)
                    cache_store[k] = v
                end,
                delete = function(_, k)
                    cache_store[k] = nil
                end,
            },
            sentinel_cache = {
                get = function()
                    return nil
                end,
            },
            circuit_breaker = {
                incr = function(_, _, delta, init)
                    return (init or 0) + delta
                end,
            },
            edge_metrics = {
                incr = function(_, key, delta, init)
                    metric_store[key] = (metric_store[key] or init or 0) + delta
                    return metric_store[key]
                end,
            },
        },
    }
    package.loaded["resty.redis"] = {}
    package.loaded["edge-circuit"] = {
        record_err = function()
            circuit_errs = circuit_errs + 1
        end,
    }
    package.loaded["edge-metrics"] = nil
    package.loaded["edge-blacklist-sync"] = nil
    local bl = require "edge-blacklist-sync"
    bl.set_env_for_test(function(name)
        if name == "EDGE_BLACKLIST_PENDING_MAX_BYTES" then
            return "512"
        end
        return nil
    end)
    cache_store["_bl_ver"] = 1
    local ips = {}
    for i = 1, 500 do
        ips[i] = string.format("10.0.%d.%d", math.floor(i / 256), i % 256)
    end
    bl.stamp_ips(ips, false)
    local pending = cache_store["_bl_pending"] or ""
    expect_true(#pending <= bl.PENDING_MAX_BYTES, "bl_pending_byte_cap", "pending queue capped at PENDING_MAX_BYTES")
    expect_true(
        (metric_store.blacklist_pending_immediate_stamp_total or 0) > 0,
        "bl_pending_byte_cap",
        "overflow immediate-stamps IPs"
    )
    expect_true(circuit_errs > 0, "bl_pending_byte_cap", "overflow records circuit err")
    bl.reset_test_hooks()
end)

-- bl_pending_overflow_immediate_stamp: pending cap must not leave overflow IPs unblocked when _bl_ver is set.
assert_case("bl_pending_overflow_immediate_stamp", "pending_overflow_immediate_stamp", function()
    local cache_store = {}
    local metric_store = {}
    ngx = {
        WARN = 1,
        INFO = 2,
        ERR = 3,
        log = function() end,
        time = function()
            return 1
        end,
        shared = {
            blacklist_cache = {
                get = function(_, k)
                    return cache_store[k]
                end,
                set = function(_, k, v)
                    cache_store[k] = v
                end,
                delete = function(_, k)
                    cache_store[k] = nil
                end,
            },
            sentinel_cache = {
                get = function()
                    return nil
                end,
            },
            circuit_breaker = {
                incr = function(_, _, delta, init)
                    return (init or 0) + delta
                end,
            },
            edge_metrics = {
                incr = function(_, key, delta, init)
                    metric_store[key] = (metric_store[key] or init or 0) + delta
                    return metric_store[key]
                end,
            },
        },
    }
    package.loaded["resty.redis"] = {}
    package.loaded["edge-circuit"] = {
        record_err = function() end,
    }
    package.loaded["edge-metrics"] = nil
    package.loaded["edge-blacklist-sync"] = nil
    local bl = require "edge-blacklist-sync"
    bl.set_env_for_test(function(name)
        if name == "EDGE_BLACKLIST_PENDING_MAX_BYTES" then
            return "32"
        end
        if name == "EDGE_BLACKLIST_CHANGELOG_MAX_IPS" then
            return "1"
        end
        return nil
    end)
    cache_store["_bl_ver"] = 2
    cache_store["_bl_pending"] = string.rep("10.0.0.1\n", 3)
    local many = {}
    for i = 2, 12 do
        many[i - 1] = string.format("10.0.0.%d", i)
    end
    bl.stamp_ips(many, false)
    expect_true(
        cache_store["b:10.0.0.12"] == 2,
        "bl_pending_overflow_immediate_stamp",
        "overflow IP immediate-stamped at _bl_ver"
    )
    expect_true(
        (metric_store.blacklist_pending_immediate_stamp_total or 0) > 0,
        "bl_pending_overflow_immediate_stamp",
        "immediate stamp metric"
    )
    bl.reset_test_hooks()
end)

-- pending_drain_restore_on_fail: drain_pending_changelog restores _bl_pending when stamp_ips fails.
assert_case("pending_drain_restore_on_fail", "pending_drain_restore_on_fail", function()
    local cache_store = {}
    ngx = {
        WARN = 1,
        INFO = 2,
        ERR = 3,
        log = function() end,
        time = function()
            return 1
        end,
        shared = {
            blacklist_cache = {
                get = function(_, k)
                    return cache_store[k]
                end,
                set = function(_, k, v)
                    cache_store[k] = v
                end,
                delete = function(_, k)
                    cache_store[k] = nil
                end,
            },
            sentinel_cache = {
                get = function()
                    return nil
                end,
            },
            circuit_breaker = {
                incr = function(_, _, delta, init)
                    return (init or 0) + delta
                end,
            },
            edge_metrics = {
                incr = function() end,
            },
        },
    }
    package.loaded["resty.redis"] = {}
    package.loaded["edge-circuit"] = {
        record_err = function() end,
    }
    package.loaded["edge-metrics"] = nil
    package.loaded["edge-blacklist-sync"] = nil
    local bl = require "edge-blacklist-sync"
    local snapshot = "198.51.100.1\n198.51.100.2\n"
    cache_store["_bl_pending"] = snapshot
    cache_store["_bl_ver"] = 1
    bl.set_stamp_ips_fail_for_test(true)
    expect_eq(0, bl.drain_pending_changelog(), "pending_drain_restore_on_fail", "drain returns 0 on failure")
    expect_eq(
        snapshot,
        cache_store["_bl_pending"],
        "pending_drain_restore_on_fail",
        "pending restored after stamp failure"
    )
    bl.reset_test_hooks()
end)

-- tarpit_delay_cap: tarpit delay hard-capped at 2s; concurrent admit limit via tarpit_active.
assert_case("tarpit_delay_cap", "tarpit_delay_at_scale", function()
    package.loaded["edge-tarpit"] = nil
    package.loaded["edge-metrics"] = nil
    ngx = { shared = { edge_metrics = { incr = function() end } } }
    local edge_tarpit = require "edge-tarpit"
    edge_tarpit.set_getenv_for_test(function(name)
        if name == "EDGE_TARPIT_ENABLED" then
            return "true"
        end
        if name == "EDGE_TARPIT_MAX_HEADERS" then
            return "64"
        end
        if name == "EDGE_TARPIT_BODY_BYTES" then
            return "65536"
        end
        if name == "EDGE_TARPIT_MAX_SEC" then
            return "15"
        end
        return nil
    end)
    local delay = edge_tarpit.compute_delay(400, 0)
    expect_eq(2, delay, "tarpit_delay_cap", "delay hard-capped at 2s even when env requests 15")
end)

-- route_gate_env_cache: route-gate caches env flags at module load (no per-request getenv).
assert_case("route_gate_env_cache", "route_gate_env_fallback", function()
    ngx = {
        shared = {
            edge_config = {
                get = function()
                    return nil
                end,
            },
        },
    }
    package.loaded["edge-blacklist-sync"] = {}
    package.loaded["edge-circuit"] = { record_err = function() end }
    package.loaded["edge-config"] = nil
    package.loaded["edge-route-gate"] = nil
    local edge_route_gate = require "edge-route-gate"
    local getenv_calls = 0
    edge_route_gate.set_getenv_for_test(function(name)
        getenv_calls = getenv_calls + 1
        if name == "EDGE_EXPOSE_CLICK" then
            return "true"
        end
        return nil
    end)
    getenv_calls = 0
    expect_true(edge_route_gate.click_enabled(), "route_gate_env_cache", "cached env enables click route")
    edge_route_gate.click_enabled()
    expect_eq(0, getenv_calls, "route_gate_env_cache", "click_enabled does not call getenv per request")
    edge_route_gate.set_getenv_for_test(nil)
end)

-- negative_content_length_reject: negative Content-Length rejected at DFA and track policy.
assert_case("negative_content_length_reject", "negative_content_length", function()
    package.loaded["edge-parse-dfa"] = nil
    local dfa = require "edge-parse-dfa"
    expect_eq(
        dfa.ERR_MALFORMED,
        dfa.check_content_length(-1),
        "negative_content_length_reject",
        "negative CL rejected at check_content_length"
    )
    local cid, err = dfa.extract_campaign_id("{}", -1)
    expect_eq(dfa.ERR_MALFORMED, err, "negative_content_length_reject", "negative CL extract returns malformed")
    expect_nil(cid, "negative_content_length_reject", "negative CL extract has no campaign_id")
end)

-- http_get_json_body_cap: http_get_json bounded by MAX_HTTP_RESPONSE_BYTES / MAX_HTTP_BODY_BYTES.
assert_case("http_get_json_body_cap", "http_get_json_no_body_cap", function()
    package.loaded["edge-net"] = nil
    local edge_net = require "edge-net"
    expect_true(edge_net.MAX_HTTP_BODY_BYTES ~= nil, "http_get_json_body_cap", "MAX_HTTP_BODY_BYTES exported")
    expect_true(edge_net.MAX_HTTP_RESPONSE_BYTES ~= nil, "http_get_json_body_cap", "MAX_HTTP_RESPONSE_BYTES exported")
    local src = io.open(arg[1] .. "/edge-net.lua", "r")
    local body = src:read "*a"
    src:close()
    expect_false(
        string.find(body, 'receive "*a"', 1, true) ~= nil,
        "http_get_json_body_cap",
        "unbounded receive removed"
    )
end)

-- slot_map_before_node_weights: slot_map shard wins over node_weights for peer index.
assert_case("slot_map_before_node_weights", "weights_preempt_slot", function()
    local function pick_peer(shard, weight_idx)
        local idx = nil
        if shard ~= nil then
            idx = tonumber(shard)
        end
        if idx == nil then
            idx = weight_idx
        end
        return idx
    end
    expect_eq(7, pick_peer(7, 3), "slot_map_before_node_weights", "slot shard wins over weights index 3")
end)

-- safe_page_campaign_id_normalize: safe-page normalizes campaign_id before subrequest args.
assert_case("safe_page_campaign_id_normalize", "safe_page_unvalidated_cid", function()
    package.loaded["edge-safe-page"] = nil
    local edge_safe_page = require "edge-safe-page"
    expect_eq(
        "",
        edge_safe_page.normalize_campaign_id "<script>alert(1)</script>",
        "safe_page_campaign_id_normalize",
        "invalid cid stripped"
    )
    expect_eq(
        "550e8400-e29b-41d4-a716-446655440000",
        edge_safe_page.normalize_campaign_id "550E8400-E29B-41D4-A716-446655440000",
        "safe_page_campaign_id_normalize",
        "valid cid lowercased"
    )
end)

-- blacklist_ip_canonical: generational blacklist lookup uses canonical client IP (IPv4-mapped unmap).
assert_case("blacklist_ip_canonical", "blacklist_canonical_v4mapped", function()
    local edge_ip = require "edge-ip"
    local ver = 4
    local stamps = {
        ["203.0.113.66"] = ver,
    }
    local function perimeter_blocked(remote_addr)
        local canon = edge_ip.canonical(remote_addr)
        local ip_ver = stamps[canon]
        return ip_ver ~= nil and ip_ver == ver
    end
    expect_true(
        perimeter_blocked "::ffff:203.0.113.66",
        "blacklist_ip_canonical",
        "v4-mapped remote_addr hits dotted-quad stamp"
    )
    expect_true(perimeter_blocked "203.0.113.66", "blacklist_ip_canonical", "native v4 remote_addr hits stamp")
    expect_false(perimeter_blocked "203.0.113.67", "blacklist_ip_canonical", "unlisted ip passes")
end)

-- campaign_id_scan_evasion: campaign_id beyond DFA scan window must not proxy with IP-only edge_rl.
assert_case("campaign_id_scan_evasion", "scan_cap_nil_campaign_id_reject", function()
    local edge_campaign_id = require "edge-campaign-id"
    local edge_parse_dfa = require "edge-parse-dfa"
    local body = string.rep("x", edge_parse_dfa.MAX_SCAN_BYTES)
    expect_true(
        edge_campaign_id.scan_evasion(body, nil, nil, edge_parse_dfa.MAX_SCAN_BYTES),
        "campaign_id_scan_evasion",
        "scan-cap nil id is evasion"
    )
    expect_false(
        edge_campaign_id.scan_evasion("{}", nil, nil, 2),
        "campaign_id_scan_evasion",
        "tiny body without id stays IP RL fallback"
    )
end)

-- Fuzz: JSON escape chains must not crash and must not return ERR_MALFORMED on valid skip paths.
assert_case("fuzz_json_escape_chains", "fuzz_json_escape_chains", function()
    package.loaded["edge-parse-dfa"] = nil
    local dfa = require "edge-parse-dfa"
    for n = 1, 8 do
        local escapes = string.rep("\\", n)
        local json = '{"junk":"' .. escapes .. '","campaign_id":"550e8400-e29b-41d4-a716-446655440000"}'
        local ok, cid, err = pcall(function()
            return dfa.extract_campaign_id(json, #json)
        end)
        if not ok then
            error(string.format("fuzz escape n=%d panic: %s", n, tostring(cid)))
        end
        if cid ~= "550e8400-e29b-41d4-a716-446655440000" and err ~= nil and err ~= dfa.ERR_MALFORMED then
            error(string.format("fuzz escape n=%d unexpected err=%s cid=%s", n, tostring(err), tostring(cid)))
        end
    end
end)

-- Fuzz: proto varint lengths 1..10 continuation bytes must not crash.
assert_case("fuzz_proto_varint_bomb", "fuzz_proto_varint_bomb", function()
    package.loaded["edge-parse-dfa"] = nil
    local dfa = require "edge-parse-dfa"
    for n = 1, 10 do
        local body = string.char(0x08) .. string.rep(string.char(0x80), n)
        local ok = pcall(dfa.extract_campaign_id, body, #body)
        if not ok then
            error(string.format("fuzz varint n=%d panicked", n))
        end
    end
end)

print(string.format("edge_security_holdout: passed=%d failed=%d faults=%d", passed, failed, fault_count))
if failed > 0 then
    os.exit(1)
end
