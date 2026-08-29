local blacklist_sync = require "edge-blacklist-sync"

local _M = {}

local dict = ngx.shared.edge_config

local DEFAULT_LIMIT = 100
local DEFAULT_WINDOW_MS = 60000
local DEFAULT_SUSPECT_PCT = 50
local DEFAULT_IVT_PCT = 10
local DEFAULT_BLOCK_PCT = 0
local DEFAULT_RETRY_SUSPECT = 30
local DEFAULT_RETRY_IVT = 60
local DEFAULT_RETRY_BLOCK = 120

function _M.get()
    local limit = dict:get "limit_per_min"
    local window_ms = dict:get "window_ms"
    if not limit or limit <= 0 then
        limit = DEFAULT_LIMIT
    end
    if not window_ms or window_ms <= 0 then
        window_ms = DEFAULT_WINDOW_MS
    end
    return limit, window_ms
end

function _M.get_tier_pct(tier)
    if tier == "suspect" then
        return dict:get "rl_pct_suspect" or DEFAULT_SUSPECT_PCT
    end
    if tier == "ivt" then
        return dict:get "rl_pct_ivt" or DEFAULT_IVT_PCT
    end
    if tier == "block" then
        return dict:get "rl_pct_block" or DEFAULT_BLOCK_PCT
    end
    return 100
end

function _M.get_retry_after(tier)
    if tier == "block" then
        return dict:get "retry_block_sec" or DEFAULT_RETRY_BLOCK
    end
    if tier == "ivt" then
        return dict:get "retry_ivt_sec" or DEFAULT_RETRY_IVT
    end
    if tier == "suspect" then
        return dict:get "retry_suspect_sec" or DEFAULT_RETRY_SUSPECT
    end
    return dict:get "retry_suspect_sec" or DEFAULT_RETRY_SUSPECT
end

function _M.get_flag(name)
    return dict:get(name)
end

local function stamp_asn_list(field, prefix)
    local raw = dict:get(field)
    if not raw or raw == "" then
        return
    end
    for asn in string.gmatch(raw, "([^,]+)") do
        asn = string.match(asn, "^%s*(.-)%s*$")
        if asn ~= "" then
            dict:set(prefix .. asn, 1)
        end
    end
end

function _M.asn_whitelisted(asn)
    if not asn or asn == "" then
        return false
    end
    asn = string.match(asn, "^%s*(.-)%s*$")
    if dict:get("asn_cdn:" .. asn) then
        return true
    end
    if dict:get("asn_mobile:" .. asn) then
        return true
    end
    return false
end

local function clear_asn_keys()
    local keys = dict:get_keys(0)
    if not keys then
        return
    end
    for _, key in ipairs(keys) do
        if string.sub(key, 1, 8) == "asn_cdn:" or string.sub(key, 1, 11) == "asn_mobile:" then
            dict:delete(key)
        end
    end
end

function _M.sync()
    local red, err = blacklist_sync.connect_any_shard()
    if not red then
        ngx.log(ngx.WARN, "edge_config: redis connect failed: ", err)
        return
    end

    local vals, cmd_err = red:hmget(
        "config:values",
        "rate_limit_per_min",
        "rate_limit_window_ms",
        "fraud_rl_suspect_pct",
        "fraud_rl_ivt_pct",
        "fraud_rl_block_pct",
        "fraud_rl_retry_suspect_sec",
        "fraud_rl_retry_ivt_sec",
        "fraud_rl_retry_block_sec",
        "asn_cdn_whitelist",
        "asn_mobile_whitelist",
        "edge_expose_click",
        "edge_expose_openrtb"
    )
    red:set_keepalive(10000, 8)
    if not vals or type(vals) ~= "table" then
        ngx.log(ngx.WARN, "edge_config: hmget failed: ", cmd_err or "empty")
        return
    end

    local limit = tonumber(vals[1])
    local window_ms = tonumber(vals[2])
    if limit and limit > 0 then
        dict:set("limit_per_min", limit)
    end
    if window_ms and window_ms > 0 then
        dict:set("window_ms", window_ms)
    end

    local suspect_pct = tonumber(vals[3])
    if suspect_pct then
        dict:set("rl_pct_suspect", suspect_pct)
    end
    local ivt_pct = tonumber(vals[4])
    if ivt_pct then
        dict:set("rl_pct_ivt", ivt_pct)
    end
    local block_pct = tonumber(vals[5])
    if block_pct then
        dict:set("rl_pct_block", block_pct)
    end

    local retry_suspect = tonumber(vals[6])
    if retry_suspect and retry_suspect > 0 then
        dict:set("retry_suspect_sec", retry_suspect)
    end
    local retry_ivt = tonumber(vals[7])
    if retry_ivt and retry_ivt > 0 then
        dict:set("retry_ivt_sec", retry_ivt)
    end
    local retry_block = tonumber(vals[8])
    if retry_block and retry_block > 0 then
        dict:set("retry_block_sec", retry_block)
    end

    clear_asn_keys()
    if vals[9] and vals[9] ~= "" then
        dict:set("asn_cdn_raw", vals[9])
        stamp_asn_list("asn_cdn_raw", "asn_cdn:")
    end
    if vals[10] and vals[10] ~= "" then
        dict:set("asn_mobile_raw", vals[10])
        stamp_asn_list("asn_mobile_raw", "asn_mobile:")
    end
    if vals[11] ~= nil then
        dict:set("edge_expose_click", vals[11])
    end
    if vals[12] ~= nil then
        dict:set("edge_expose_openrtb", vals[12])
    end
end

return _M
