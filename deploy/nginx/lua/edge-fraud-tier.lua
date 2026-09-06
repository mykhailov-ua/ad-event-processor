-- Map trusted fraud score (0-100) to edge rate-limit tier bucket names.
-- Runtime: all workers access phase; pure functions + trusted header read; no ngx.shared.
--
-- Consumers: edge-rl.lua tier_limit(), retry_after_sec(); edge_track_policy apply_campaign_rl block branch.
-- Score source: EDGE_TRUSTED_FRAUD_SCORE_HEADER (default X-Edge-Trusted-Fraud-Score) only.
-- Client X-Fraud-Score is cleared before read (client_fraud_score_rl_tier); not used for edge RL or tier block.
--
-- Cache invalidation: none.
--
-- Tier thresholds (score inclusive upper bound):
-- - pass: score <= PASS_MAX 30.
-- - suspect: score <= SUSPECT_MAX 60.
-- - ivt: score <= IVT_MAX 80.
-- - block: score > IVT_MAX.
--
-- Missing trusted score: score_for_edge_rl returns PASS_MAX+1 (suspect tier RL); not full pass tier.
--
-- Constants and limits:
-- - PASS_MAX 30, SUSPECT_MAX 60, IVT_MAX 80.
-- - tier_from_score returns (tier, clamped_score).
--
-- Forbidden: reading http_x_fraud_score on public edge; treating edge tier block as tracker FilterEngine.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-fraud-tier.lua
-- bash scripts/test/edge/lua_tests.sh unit
local _M = {}

local PASS_MAX = 30
local SUSPECT_MAX = 60
local IVT_MAX = 80

local getenv = os.getenv

local DEFAULT_TRUSTED_FRAUD_SCORE_HEADER = "X-Edge-Trusted-Fraud-Score"
local UNTRUSTED_FRAUD_SCORE_HEADER = "X-Fraud-Score"

local function reload_trusted_header()
    local name = getenv "EDGE_TRUSTED_FRAUD_SCORE_HEADER" or DEFAULT_TRUSTED_FRAUD_SCORE_HEADER
    if name == "" then
        name = DEFAULT_TRUSTED_FRAUD_SCORE_HEADER
    end
    return name
end

local TRUSTED_FRAUD_SCORE_HEADER = reload_trusted_header()

local function header_http_var(name)
    name = string.lower(name)
    name = string.gsub(name, "-", "_")
    return "http_" .. name
end

local TRUSTED_FRAUD_SCORE_VAR = header_http_var(TRUSTED_FRAUD_SCORE_HEADER)

function _M.set_getenv_for_test(fn)
    if fn then
        getenv = fn
    else
        getenv = os.getenv
    end
    TRUSTED_FRAUD_SCORE_HEADER = reload_trusted_header()
    TRUSTED_FRAUD_SCORE_VAR = header_http_var(TRUSTED_FRAUD_SCORE_HEADER)
end

function _M.trusted_header_name()
    return TRUSTED_FRAUD_SCORE_HEADER
end

function _M.sanitize_untrusted_fraud_score_headers()
    if TRUSTED_FRAUD_SCORE_HEADER ~= UNTRUSTED_FRAUD_SCORE_HEADER then
        ngx.req.clear_header(UNTRUSTED_FRAUD_SCORE_HEADER)
    end
end

function _M.trusted_fraud_score()
    local raw = ngx.var[TRUSTED_FRAUD_SCORE_VAR]
    if not raw or raw == "" then
        return nil
    end
    return tonumber(raw)
end

function _M.score_for_edge_rl()
    local score = _M.trusted_fraud_score()
    if score == nil then
        return PASS_MAX + 1
    end
    return score
end

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
