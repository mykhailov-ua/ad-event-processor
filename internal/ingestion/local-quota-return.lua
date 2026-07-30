
local amt = tonumber(ARGV[1]) or 0
if amt <= 0 then
    return tonumber(redis.call("GET", KEYS[1]) or "0")
end
return redis.call("INCRBY", KEYS[1], amt)
