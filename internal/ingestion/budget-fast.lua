-- Global cache
local redis_call = redis.call
local tonumber = tonumber

local batch = redis_call("MGET", KEYS[1], KEYS[2], KEYS[8], KEYS[9])
local spend = batch[1]
local idem = batch[2]
local fence = batch[3]
local frozen = batch[4]

if not spend then
    return -1
end

if idem then
    return 0
end

local redis_epoch = tonumber(fence) or 0
local routing_epoch = tonumber(ARGV[13]) or 0
if (redis_epoch > 0 and redis_epoch ~= routing_epoch) or frozen then
    return 11
end

local client_ip = ARGV[9] or ""
local fraud_list_hit = false
if client_ip ~= "" and KEYS[10] and KEYS[10] ~= "fcap:ignored" and redis_call("SISMEMBER", KEYS[10], client_ip) == 1 then
    fraud_list_hit = true
end

local placement_id = ARGV[16] or ""
if placement_id ~= "" and KEYS[11] and KEYS[11] ~= "fcap:ignored" and redis_call("HEXISTS", KEYS[11], placement_id) == 1 then
    return 14
end

local max_rpd = tonumber(ARGV[14]) or 0
if max_rpd > 0 and KEYS[12] and KEYS[12] ~= "fcap:ignored" then
    local ingress = redis_call("INCR", KEYS[12])
    if ingress == 1 then
        redis_call("EXPIRE", KEYS[12], tonumber(ARGV[15]) or 100800)
    end
    if ingress > max_rpd then
        return 12
    end
end

local amount = tonumber(ARGV[1]) or 0
local skip_budget = ARGV[12] == "1"

if not skip_budget then
    local budget = tonumber(spend) or 0
    if budget < amount then
        return 3
    end

    redis_call("INCRBY", KEYS[1], -amount)

    local c_sync = redis_call("INCRBY", KEYS[3], amount)
    if c_sync == amount then
        redis_call("SADD", KEYS[5], ARGV[3])
    end

    local cust_sync = redis_call("INCRBY", KEYS[4], amount)
    if cust_sync == amount then
        redis_call("SADD", KEYS[6], ARGV[4])
    end

    local dw = redis_call("GET", "slot_migration:dual_write")
    if dw then
        redis_call("XADD", "slot_migration:delta", "MAXLEN", "~", "100000", "*",
            "campaign_id", ARGV[3],
            "amount", ARGV[1],
            "spend_key", KEYS[1])
    end
end

redis_call("SET", KEYS[2], "1", "EX", ARGV[2])

if KEYS[7] and KEYS[7] ~= "fcap:ignored" and KEYS[7] ~= "" then
    redis_call("XADD", KEYS[7], "MAXLEN", "~", ARGV[5], "*",
        "click_id", ARGV[6],
        "campaign_id", ARGV[3],
        "user_id", ARGV[11],
        "type", ARGV[7],
        "payload", ARGV[8],
        "ip", ARGV[9],
        "ua", ARGV[10])
end

if fraud_list_hit then
    return 21
end
return 0
