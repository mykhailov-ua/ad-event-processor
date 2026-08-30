-- Map X-Fraud-Score request header (0-100) to edge rate-limit tier bucket names.
-- Runtime: all workers access phase; pure function, no ngx.shared.
--
-- Consumers: edge-rl.lua tier_limit(), retry_after_sec(); edge_track_policy apply_campaign_rl block branch.
-- Score source: X-Fraud-Score header (edge_track_policy fraud_score_from_headers); not tracker ML inference.
--
-- Cache invalidation: none.
--
-- Tier thresholds (score inclusive upper bound):
-- - pass: score <= PASS_MAX 30.
-- - suspect: score <= SUSPECT_MAX 60.
-- - ivt: score <= IVT_MAX 80.
-- - block: score > IVT_MAX.
--
-- State machine: clamp score 0..100 -> tier string + numeric score for RL key suffix.
--
-- Constants and limits:
-- - PASS_MAX 30, SUSPECT_MAX 60, IVT_MAX 80.
-- - tier_from_score returns (tier, clamped_score).
--
-- Does not enforce tracker FilterEngine, Redis fraud actions, silent_reject, or campaign PG mutation.
--
-- Forbidden: treating edge tier block as silent_reject or per-IP Redis blacklist on tracker.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-fraud-tier.lua
-- bash scripts/test/edge/lua_tests.sh
local _M = {}

local PASS_MAX = 30
local SUSPECT_MAX = 60
local IVT_MAX = 80

function _M.tier_from_score(score)
    local n = tonumber(score) or 0
    if n < 0 then
        n = 0
    elseif n > 100 then
        n = 100
    end
    if n <= PASS_MAX then
        return "pass", n
    end
    if n <= SUSPECT_MAX then
        return "suspect", n
    end
    if n <= IVT_MAX then
        return "ivt", n
    end
    return "block", n
end

return _M
