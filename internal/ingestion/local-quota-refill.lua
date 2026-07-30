
local q = tonumber(redis.call("GET", KEYS[1]) or "0")
local chunk = tonumber(ARGV[1]) or 0
if chunk <= 0 then
    return -1
end
if q < chunk then
    return -1
end
redis.call("DECRBY", KEYS[1], chunk)
return chunk
