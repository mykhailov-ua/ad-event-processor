-- Per-campaign edge rate limiter with fraud-tier scaling in ngx.shared.edge_rl.
-- Runtime: all workers access phase via edge_track_policy.apply_campaign_rl (not tracker UnifiedFilter debit).
--
-- Consumers: edge_track_policy.lua allow(), retry_after_sec(); tier from edge-fraud-tier.lua.
-- Limits from edge-config.get() mirror (limit_per_min, window_ms, rl_pct_*, retry_*_sec).
--
-- Cache invalidation: TTL bucket keys "{campaign_id}:{tier}:{bucket}" via incr(..., 0, window_sec*2).
-- Old buckets expire after 2x window; no explicit delete. incr failure: fail-closed (deny request).
--
-- ngx.shared edge_rl keys (number count per key):
-- - {campaign_id}:{tier}:{bucket} where tier is pass|suspect|ivt|block; bucket = floor(ngx.time()/window_sec).
--
-- State machine (per request):
-- - nil/empty campaign_id -> allow (no SHM touch).
-- - base_limit <= 0 from config -> allow.
-- - tier block or scaled limit <= 0 -> deny (edge_track_policy -> 403 fraud block).
-- - incr count <= limit -> allow; else deny (429 with Retry-After from edge-config tier retry).
--
-- Constants and limits:
-- - window_sec = max(1, floor(window_ms / 1000)); incr TTL = window_sec * 2.
-- - tier_limit: pct <= 0 -> limit 0; pct >= 100 -> base_limit; else max(1, floor(base_limit * pct / 100)).
-- - fraud_score clamp and tier thresholds owned by edge-fraud-tier.lua.
--
-- Forbidden: assuming edge RL replaces Redis campaign budget, UnifiedFilter debit, or tracker fraud blacklist.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-rl.lua
-- bash scripts/test/edge/lua_tests.sh
local edge_config = require "edge-config"
local edge_fraud_tier = require "edge-fraud-tier"

local _M = {}

local rl_dict = ngx.shared.edge_rl

local function tier_limit(base_limit, fraud_score)
    local tier = edge_fraud_tier.tier_from_score(fraud_score)
    local pct = edge_config.get_tier_pct(tier)
    if pct <= 0 then
        return 0, tier
    end
    if pct >= 100 then
        return base_limit, tier
    end
    local scaled = math.max(1, math.floor(base_limit * pct / 100))
    return scaled, tier
end

function _M.retry_after_sec(fraud_score)
    local tier = edge_fraud_tier.tier_from_score(fraud_score)
    return edge_config.get_retry_after(tier)
end

-- Per-campaign edge_rl incr: key {campaign_id}:{tier}:{bucket}; TTL 2x window_sec. incr fail -> deny (fail-closed).
-- Tier scaling from edge-config rl_pct_*; does not debit Redis campaign budget (tracker UnifiedFilter owns spend).
function _M.allow(campaign_id, fraud_score)
    if not campaign_id or campaign_id == "" then
        return true
    end

    local base_limit, window_ms = edge_config.get()
    if base_limit <= 0 then
        return true
    end

    local limit, tier = tier_limit(base_limit, fraud_score or 0)
    if tier == "block" or limit <= 0 then
        return false
    end

    local window_sec = math.max(1, math.floor(window_ms / 1000))
    local bucket = math.floor(ngx.time() / window_sec)
    local key = campaign_id .. ":" .. tier .. ":" .. tostring(bucket)

    local count, err = rl_dict:incr(key, 1, 0, window_sec * 2)
    if not count then
        ngx.log(ngx.ERR, "edge_rl: incr failed: ", err)
        return false
    end

    return count <= limit
end

return _M
