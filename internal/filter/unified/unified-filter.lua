-- Unified campaign debit, dedup, pacing, fcap, optional stream XADD (single Redis EVALSHA).
-- Go wire: internal/filter/unified/unified.go runUnifiedLua; KEYS layout in doc.go.
-- Verify: go test ./internal/filter/unified/... -short -count=1

local redis_call = redis.call
local tonumber = tonumber
local math_floor = math.floor

-- ARGV[25] quota mode; ARGV[26..27] local-quota refill chunk and threshold pct.
local quota_enabled = ARGV[25] == "1"
local chunk_size = tonumber(ARGV[26]) or 0
local refill_threshold_pct = tonumber(ARGV[27]) or 20

-- ARGV[29] filter deadline monotonic ns; ARGV[30] now monotonic ns; ARGV[31] degrade_ns (default 2ms).
-- degraded=true skips pacing, fcap, TTC, and impression-ts writes but still debits when budget path runs.
local deadline_ns = tonumber(ARGV[29]) or 0
local now_mono_ns = tonumber(ARGV[30]) or 0
local degrade_ns = tonumber(ARGV[31]) or 2000000
local degraded = deadline_ns > 0 and now_mono_ns > 0 and (deadline_ns - now_mono_ns) < degrade_ns

local b
local idem_exists
local daily_spent_raw
local fcap_raw
-- KEYS[13] quota ledger when ARGV[25]=1; else KEYS[3] campaign budget micro-units.
local spend_key = quota_enabled and KEYS[13] or KEYS[3]

-- MGET: spend_key, KEYS[4] idempotency marker, KEYS[10] daily spend, KEYS[11] user fcap counter.
local batch = redis_call("MGET", spend_key, KEYS[4], KEYS[10], KEYS[11])
b = batch[1]
idem_exists = batch[2]
daily_spent_raw = batch[3]
fcap_raw = batch[4]

if not b then
    return -1
end

-- Prior accept already set KEYS[4]; return 0 without debit (distinct from dedup SET NX return 2).
if idem_exists then
    return 0
end

-- KEYS[16] migration routing epoch; KEYS[17] budget frozen flag. Return 11: no debit past fence.
local barriers = redis_call("MGET", KEYS[16], KEYS[17])
local redis_epoch = tonumber(barriers[1]) or 0
local routing_epoch = tonumber(ARGV[28]) or 0
if (redis_epoch > 0 and redis_epoch ~= routing_epoch) or barriers[2] then
    return 11
end

local evt_type = ARGV[10] or ""
local ttc_min_ms = tonumber(ARGV[20]) or 0
local now_ms = tonumber(ARGV[21]) or 0
local ttc_fail_closed = ARGV[23] == "1"
-- ARGV[34]=1 when Go LocalTTCCache handles TTC; Lua skips KEYS[12] read on clicks.
local ttc_in_go = ARGV[34] == "1"
local ttc_bypass = false

if not degraded and not ttc_in_go and evt_type == "click" and ttc_min_ms > 0 then
    local imp_ts_raw = redis_call("GET", KEYS[12])
    if imp_ts_raw then
        local imp_ts = tonumber(imp_ts_raw)
        if imp_ts and (now_ms - imp_ts) < ttc_min_ms then
            return 6
        end
    elseif ttc_fail_closed then
        return 7
    else
        -- Missing impression ts with fail-open: accept path continues; return 10 after side effects.
        ttc_bypass = true
    end
end

-- ARGV[24]=1: skip INCRBY spend and sync counters (fraud-signal-only or replay-safe paths).
local skip_budget = ARGV[24] == "1"

local budget = tonumber(b) or 0
local amount = tonumber(ARGV[4]) or 0
local freq_limit = tonumber(ARGV[18]) or 0
local user_id = ARGV[17] or ""

if not skip_budget then
    if budget < amount then
        return 3
    end

    if not degraded and ARGV[14] == "1" then
        local daily_spent = tonumber(daily_spent_raw or 0)
        local daily_limit = tonumber(ARGV[15]) or 0
        local hour_num = tonumber(ARGV[16]) or 24
        local cumulative_limit = math_floor((daily_limit * hour_num) / 24)

        if daily_spent + amount > cumulative_limit then
            return 4
        end
    end
end

if not degraded and freq_limit > 0 and user_id ~= "" then
    local current_fcap = tonumber(fcap_raw or 0)
    if current_fcap >= freq_limit then
        return 5
    end
end

-- KEYS[2] event dedup NX; return 2 on race duplicate (budget gates already passed, no second debit).
local is_dup = redis_call("SET", KEYS[2], "NX", "EX", ARGV[3])
if not is_dup then
    return 2
end

if not skip_budget then
    redis_call("INCRBY", spend_key, -amount)
    -- KEYS[5]/KEYS[7] campaign sync counter and dirty set; SADD only on first increment in window.
    local c_sync = redis_call("INCRBY", KEYS[5], amount)
    if c_sync == amount then
        redis_call("SADD", KEYS[7], ARGV[6])
    end

    local cust_sync = redis_call("INCRBY", KEYS[6], amount)
    if cust_sync == amount then
        redis_call("SADD", KEYS[8], ARGV[7])
    end

    local dw = redis_call("GET", "slot_migration:dual_write")
    if dw then
        redis_call(
            "XADD",
            "slot_migration:delta",
            "MAXLEN",
            "~",
            "100000",
            "*",
            "campaign_id",
            ARGV[6],
            "amount",
            ARGV[4],
            "spend_key",
            spend_key
        )
    end
end

redis_call("SET", KEYS[4], "1", "EX", ARGV[5])

if not skip_budget and not degraded and ARGV[14] == "1" then
    local ds = redis_call("INCRBY", KEYS[10], amount)
    if ds == amount then
        redis_call("EXPIRE", KEYS[10], 172800)
    end
end

if not degraded and freq_limit > 0 and user_id ~= "" then
    local new_fcap = redis_call("INCR", KEYS[11])
    if new_fcap == 1 then
        redis_call("EXPIRE", KEYS[11], tonumber(ARGV[19]))
    end
end

if not degraded and evt_type == "impression" then
    local imp_ts_ttl = tonumber(ARGV[22]) or 600
    redis_call("SET", KEYS[12], now_ms, "EX", imp_ts_ttl)
end

local payload = ARGV[11] or ""
local campaign_id = ARGV[6] or ""
local click_id = ARGV[9] or ""
if campaign_id ~= "" and click_id ~= "" then
    local tg_meta = redis_call("GET", "{" .. campaign_id .. "}tg:click:" .. click_id)
    if tg_meta then
        payload = tg_meta
    end
end

-- KEYS[9] stream; fcap:ignored or empty skips XADD (Go StreamProducer is sole writer when deferred).
if KEYS[9] and KEYS[9] ~= "fcap:ignored" and KEYS[9] ~= "" then
    redis_call(
        "XADD",
        KEYS[9],
        "MAXLEN",
        "~",
        ARGV[8],
        "*",
        "click_id",
        click_id,
        "campaign_id",
        campaign_id,
        "user_id",
        user_id,
        "type",
        evt_type,
        "payload",
        payload,
        "ip",
        ARGV[12],
        "ua",
        ARGV[13]
    )
end

if not degraded and quota_enabled and chunk_size > 0 and spend_key == KEYS[13] and not skip_budget then
    local quota_after = budget - amount
    local threshold = math_floor(chunk_size * refill_threshold_pct / 100)
    if quota_after < threshold then
        -- KEYS[14] refill lock NX 10s; KEYS[15] refill_needed set for background refill worker.
        local locked = redis_call("SET", KEYS[14], "1", "NX", "EX", 10)
        if locked then
            redis_call("SADD", KEYS[15], ARGV[6])
        end
    end
end

if ttc_bypass then
    -- Return 10: accept after debit; missing impression ts with fail-open (metrics.TTCBypassTotal in Go).
    return 10
end
if degraded then
    -- Return 20: accept after debit; remaining deadline < degrade_ns; pacing/fcap/TTC paths skipped.
    return 20
end
return 0
