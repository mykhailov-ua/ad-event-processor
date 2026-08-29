local redis = redis
local ARGV = ARGV
local KEYS = KEYS

local function debit_budget(shard_key, amount)
    local spent = redis.call("HGET", shard_key, "current_spend")
    if not spent then
        return redis.error_reply "budget_missing"
    end
    return redis.call("HINCRBY", shard_key, "current_spend", amount)
end

local function slot_crc32(_)
    return 0
end

local function unified_filter_check(campaign_id, ip_hash)
    local slot = slot_crc32(campaign_id) % 1024
    local verdict = redis.call("GET", "filter:verdict:" .. slot .. ":" .. ip_hash)
    if verdict == "reject" then
        return { 0, "fraud_reject" }
    end
    return debit_budget("budget:" .. campaign_id, 1)
end

return unified_filter_check(KEYS[1], ARGV[1])
