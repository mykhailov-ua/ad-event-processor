-- Probe cluster observe (Redis Lua).
-- KEYS[1] meta hash; KEYS[2] sessions set; KEYS[3] campaigns set
-- ARGV[1] session_id; ARGV[2] campaign_id; ARGV[3] probe_score; ARGV[4] verify_incr
-- ARGV[5] ttl_sec; ARGV[6] ja3; ARGV[7] ja4; ARGV[8] tcp_sig; ARGV[9] webgl_hex
-- Returns: session_count, campaign_count, verify_count, avg_probe_score
local new_sess = redis.call("SADD", KEYS[2], ARGV[1])
if new_sess == 1 then
    redis.call("HINCRBY", KEYS[1], "session_count", 1)
end
local new_camp = redis.call("SADD", KEYS[3], ARGV[2])
if new_camp == 1 then
    redis.call("HINCRBY", KEYS[1], "campaign_count", 1)
end
if tonumber(ARGV[4]) == 1 then
    redis.call("HINCRBY", KEYS[1], "verify_count", 1)
end
local ps = tonumber(ARGV[3])
if ps > 0 then
    redis.call("HINCRBY", KEYS[1], "probe_score_sum", ps)
    redis.call("HINCRBY", KEYS[1], "probe_score_n", 1)
end
if ARGV[6] ~= "" then
    redis.call("HSET", KEYS[1], "ja3", ARGV[6])
end
if ARGV[7] ~= "" then
    redis.call("HSET", KEYS[1], "ja4", ARGV[7])
end
if ARGV[8] ~= "" then
    redis.call("HSET", KEYS[1], "tcp_sig", ARGV[8])
end
if ARGV[9] ~= "" then
    redis.call("HSET", KEYS[1], "webgl", ARGV[9])
end
local ttl = tonumber(ARGV[5])
redis.call("EXPIRE", KEYS[1], ttl)
redis.call("EXPIRE", KEYS[2], ttl)
redis.call("EXPIRE", KEYS[3], ttl)
local sc = tonumber(redis.call("HGET", KEYS[1], "session_count") or "0")
local cc = tonumber(redis.call("HGET", KEYS[1], "campaign_count") or "0")
local vc = tonumber(redis.call("HGET", KEYS[1], "verify_count") or "0")
local sum = tonumber(redis.call("HGET", KEYS[1], "probe_score_sum") or "0")
local n = tonumber(redis.call("HGET", KEYS[1], "probe_score_n") or "0")
local avg = 0
if n > 0 then
    avg = math.floor(sum / n)
end
return { sc, cc, vc, avg }
