-- Budget debit rollback after post-debit enqueue failure (Redis Lua).
-- Runs on ingest RollbackRedisDebit; restores spend key, sync counters, clears click idempotency.
-- KEYS[1] budget spend key; KEYS[2] idempotency:click; KEYS[3] campaign sync; KEYS[4] customer sync;
-- KEYS[5] dirty campaigns set; KEYS[6] dirty customers set; KEYS[7] rollback:click guard (SET NX).
-- ARGV[1] refund amount micro-units; ARGV[2] campaign id string; ARGV[3] customer id string;
-- ARGV[4] rollback guard TTL seconds (int).
-- Returns: 0 invalid amount; 1 rolled back; 2 duplicate rollback (idempotent no-op).
--
-- Verify:
-- bash scripts/ci/static/budget_rollback_gate.sh
local redis_call = redis.call
local amount = tonumber(ARGV[1]) or 0
if amount <= 0 then
    return 0
end

local guard_ttl = tonumber(ARGV[4]) or 3600
if redis_call("SET", KEYS[7], "1", "NX", "EX", guard_ttl) == false then
    return 2
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
