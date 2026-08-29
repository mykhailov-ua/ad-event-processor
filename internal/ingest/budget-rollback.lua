local redis_call = redis.call
local amount = tonumber(ARGV[1]) or 0
if amount <= 0 then
    return 0
end

redis_call("INCRBY", KEYS[1], amount)

local c_sync = redis_call("INCRBY", KEYS[3], -amount)
if c_sync <= 0 then
    redis_call("DEL", KEYS[3])
    redis_call("SREM", KEYS[5], ARGV[2])
end

local cust_sync = redis_call("INCRBY", KEYS[4], -amount)
if cust_sync <= 0 then
    redis_call("DEL", KEYS[4])
    redis_call("SREM", KEYS[6], ARGV[3])
end

redis_call("DEL", KEYS[2])

return 1
