-- Zero-copy body DFA: extract campaign_id from JSON, OpenRTB 3, or native protobuf wire.
-- Runtime: all workers access phase via edge_track_policy (no ngx.shared; pure scan on body chunk).
-- Must match Go ParseTrackRequestJSON* / vtproto budgets (parser.mdc).
--
-- Consumers: edge_track_policy sets ngx.ctx.campaign_id; edge-slot-map.get_shard uses same id bytes.
-- edge-shard-balancer routes slot = CRC32C(campaign_id) & 1023.
--
-- Cache invalidation: none (no SHM).
--
-- Return codes (string):
-- - ERR_OVERSIZE: CL or field exceeds budget.
-- - ERR_MALFORMED: wire/JSON shape break inside scan window.
-- - nil campaign_id: allowed when id not found in scan window (may proxy without shard affinity).
--
-- State machine: check_content_length -> extract_campaign_id(body, cl, schema) -> format UUID or raw id.
--
-- Constants and limits (must match tracker):
-- - MAX_BODY_BYTES 1048576 (1 MiB Content-Length cap).
-- - MAX_SCAN_BYTES 8192 scan/peek window.
-- - MAX_CAMPAIGN_LEN 64; MAX_FIELD_LEN 65536.
-- - TRACKER_INGRESS_SCHEMA ad_event_processor_native (protobuf field 1) or openrtb_3 (item[0].id JSON).
-- - scan_limit_for: min(body_len, content_length, MAX_SCAN_BYTES).
-- - protobuf varint shift cap 35 bits; 16-byte binary UUID expanded to hyphenated hex.
--
-- HTTP mapping (via edge_track_policy): ERR_OVERSIZE -> 413; malformed may proxy with nil campaign_id.
--
-- Forbidden: full JSON decode; allocations beyond scan budget; schema drift from http1IngressCanonical.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-parse-dfa.lua
-- bash scripts/test/edge/lua_tests.sh edge_parse_dfa_fault
-- go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
local bit = require "bit"

local _M = {}

_M.MAX_BODY_BYTES = 1048576
_M.MAX_SCAN_BYTES = 8192
_M.MAX_CAMPAIGN_LEN = 64
_M.MAX_FIELD_LEN = 65536

local MAX_BODY_BYTES = _M.MAX_BODY_BYTES
local MAX_SCAN_BYTES = _M.MAX_SCAN_BYTES
local MAX_CAMPAIGN_LEN = _M.MAX_CAMPAIGN_LEN
local MAX_FIELD_LEN = _M.MAX_FIELD_LEN

local ERR_OVERSIZE = "oversize"
local ERR_MALFORMED = "malformed"

local byte = string.byte
local char = string.char
local sub = string.sub

local HEX = "0123456789abcdef"

local function byte_to_hex(b)
    local hi = bit.rshift(b, 4) + 1
    local lo = bit.band(b, 0x0f) + 1
    return char(byte(HEX, hi), byte(HEX, lo))
end

local function format_campaign_id(raw)
    if not raw or raw == "" then
        return nil, nil
    end
    if #raw == 16 then
        local g = function(i)
            return byte_to_hex(byte(raw, i))
        end
        return table.concat {
            g(1),
            g(2),
            g(3),
            g(4),
            "-",
            g(5),
            g(6),
            "-",
            g(7),
            g(8),
            "-",
            g(9),
            g(10),
            "-",
            g(11),
            g(12),
            g(13),
            g(14),
            g(15),
            g(16),
        },
            nil
    end
    if #raw > MAX_CAMPAIGN_LEN then
        return nil, ERR_OVERSIZE
    end
    return raw, nil
end

local function decode_varint(data, pos, limit)
    local val = 0
    local shift = 0
    while pos <= limit do
        local b = byte(data, pos)
        if not b then
            return nil, nil, ERR_MALFORMED
        end
        pos = pos + 1
        val = val + bit.lshift(bit.band(b, 0x7f), shift)
        if bit.band(b, 0x80) == 0 then
            return val, pos, nil
        end
        shift = shift + 7
        if shift >= 35 then
            return nil, nil, ERR_MALFORMED
        end
    end
    return nil, nil, ERR_MALFORMED
end

local function scan_limit_for(body_len, content_length)
    local limit = body_len
    if content_length and content_length > 0 and content_length < limit then
        limit = content_length
    end
    if limit > MAX_SCAN_BYTES then
        limit = MAX_SCAN_BYTES
    end
    return limit
end

local function scan_proto_dfa(data, scan_limit)
    local pos = 1
    while pos <= scan_limit do
        local tag_b = byte(data, pos)
        if not tag_b then
            break
        end
        pos = pos + 1
        local wire = bit.band(tag_b, 0x07)
        local field = bit.rshift(tag_b, 3)

        if wire == 0 then
            local _, next_pos, err = decode_varint(data, pos, scan_limit)
            if err or not next_pos then
                return nil, err or ERR_MALFORMED
            end
            pos = next_pos
        elseif wire == 1 then
            if pos + 7 > scan_limit then
                return nil, ERR_MALFORMED
            end
            pos = pos + 8
        elseif wire == 2 then
            local field_len, new_pos, err = decode_varint(data, pos, scan_limit)
            if err then
                return nil, err
            end
            if not field_len or not new_pos or field_len > MAX_FIELD_LEN then
                return nil, ERR_OVERSIZE
            end
            if new_pos + field_len - 1 > scan_limit then
                return nil, ERR_MALFORMED
            end
            if field == 1 then
                if field_len > MAX_CAMPAIGN_LEN then
                    return nil, ERR_OVERSIZE
                end
                local raw = sub(data, new_pos, new_pos + field_len - 1)
                return format_campaign_id(raw)
            end
            pos = new_pos + field_len
        elseif wire == 5 then
            if pos + 3 > scan_limit then
                return nil, ERR_MALFORMED
            end
            pos = pos + 4
        else
            return nil, ERR_MALFORMED
        end
    end
    return nil, nil
end

local function json_key_id(key)
    if string.find(key, "\\", 1, true) then
        return nil
    end
    local n = #key
    if n == 11 and sub(key, 1, 8) == "campaign" and sub(key, 9, 11) == "_id" then
        return "campaign_id"
    end
    if n == 4 and key == "item" then
        return "item"
    end
    if n == 2 and key == "id" then
        return "id"
    end
    return nil
end

local function is_json_ws(b)
    return b == 32 or b == 9 or b == 10 or b == 13
end

local function skip_json_value(data, pos, scan_limit)
    local err
    local b = byte(data, pos)
    if not b then
        return nil, ERR_MALFORMED
    end
    if b == 34 then
        pos = pos + 1
        while pos <= scan_limit do
            local c = byte(data, pos)
            if not c then
                return nil, ERR_MALFORMED
            end
            if c == 34 then
                return pos + 1, nil
            end
            if c == 92 then
                pos = pos + 2
            else
                pos = pos + 1
            end
        end
        return nil, ERR_MALFORMED
    end
    if b == 123 then
        pos = pos + 1
        while pos <= scan_limit do
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            if byte(data, pos) == 125 then
                return pos + 1, nil
            end
            if byte(data, pos) ~= 34 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
            while pos <= scan_limit and byte(data, pos) ~= 34 do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit or byte(data, pos) ~= 58 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
            local next_pos
            next_pos, err = skip_json_value(data, pos, scan_limit)
            if not next_pos then
                return nil, err
            end
            pos = next_pos
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            local sep = byte(data, pos)
            if sep == 125 then
                return pos + 1, nil
            end
            if sep ~= 44 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
        end
        return nil, ERR_MALFORMED
    end
    if b == 91 then
        pos = pos + 1
        while pos <= scan_limit do
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            if byte(data, pos) == 93 then
                return pos + 1, nil
            end
            local next_pos
            next_pos, err = skip_json_value(data, pos, scan_limit)
            if not next_pos then
                return nil, err
            end
            pos = next_pos
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            local sep = byte(data, pos)
            if sep == 93 then
                return pos + 1, nil
            end
            if sep ~= 44 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
        end
        return nil, ERR_MALFORMED
    end
    if (b >= 48 and b <= 57) or b == 45 then
        pos = pos + 1
        while pos <= scan_limit do
            local c = byte(data, pos)
            if not c then
                break
            end
            if not ((c >= 48 and c <= 57) or c == 46 or c == 101 or c == 69 or c == 43 or c == 45) then
                break
            end
            pos = pos + 1
        end
        return pos, nil
    end
    if b == 116 then
        if sub(data, pos, pos + 3) == "true" then
            return pos + 4, nil
        end
        return nil, ERR_MALFORMED
    end
    if b == 102 then
        if sub(data, pos, pos + 4) == "false" then
            return pos + 5, nil
        end
        return nil, ERR_MALFORMED
    end
    if b == 110 then
        if sub(data, pos, pos + 3) == "null" then
            return pos + 4, nil
        end
        return nil, ERR_MALFORMED
    end
    return nil, ERR_MALFORMED
end

local function read_json_string(data, pos, scan_limit)
    if byte(data, pos) ~= 34 then
        return nil, nil, ERR_MALFORMED
    end
    pos = pos + 1
    local val_start = pos
    while pos <= scan_limit do
        local c = byte(data, pos)
        if not c then
            return nil, nil, ERR_MALFORMED
        end
        if c == 34 then
            local raw = sub(data, val_start, pos - 1)
            if #raw > MAX_CAMPAIGN_LEN then
                return nil, nil, ERR_OVERSIZE
            end
            if string.find(raw, "\0", 1, true) then
                return nil, nil, ERR_MALFORMED
            end
            return raw, pos + 1, nil
        end
        if c == 0 then
            return nil, nil, ERR_MALFORMED
        end
        if c == 92 then
            pos = pos + 2
        else
            pos = pos + 1
        end
    end
    return nil, nil, ERR_MALFORMED
end

local function scan_json_dfa(data, scan_limit)
    local err
    local last_cid = nil
    local pos = 1
    while pos <= scan_limit and is_json_ws(byte(data, pos)) do
        pos = pos + 1
    end
    if pos > scan_limit or byte(data, pos) ~= 123 then
        return nil, ERR_MALFORMED
    end
    pos = pos + 1

    while pos <= scan_limit do
        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if pos > scan_limit then
            return nil, ERR_MALFORMED
        end
        if byte(data, pos) == 125 then
            return last_cid, nil
        end
        if byte(data, pos) ~= 34 then
            return nil, ERR_MALFORMED
        end
        pos = pos + 1
        local key_start = pos
        while pos <= scan_limit and byte(data, pos) ~= 34 do
            if byte(data, pos) == 92 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
        end
        if pos > scan_limit then
            return nil, ERR_MALFORMED
        end
        local key = sub(data, key_start, pos - 1)
        pos = pos + 1

        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if pos > scan_limit or byte(data, pos) ~= 58 then
            return nil, ERR_MALFORMED
        end
        pos = pos + 1
        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if pos > scan_limit then
            return nil, ERR_MALFORMED
        end

        local kid = json_key_id(key)
        if kid == "campaign_id" then
            local raw
            raw, pos, err = read_json_string(data, pos, scan_limit)
            if not raw then
                return nil, err
            end
            last_cid = raw
        else
            local next_pos
            next_pos, err = skip_json_value(data, pos, scan_limit)
            if not next_pos then
                return nil, err
            end
            pos = next_pos
        end

        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if pos > scan_limit then
            return nil, ERR_MALFORMED
        end
        local sep = byte(data, pos)
        if sep == 125 then
            return last_cid, nil
        end
        if sep ~= 44 then
            return nil, ERR_MALFORMED
        end
        pos = pos + 1
    end
    return nil, ERR_MALFORMED
end

local function scan_item_object(data, pos, scan_limit)
    local err
    if byte(data, pos) ~= 123 then
        return nil, nil, ERR_MALFORMED
    end
    pos = pos + 1
    local item_id = nil
    while pos <= scan_limit do
        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if pos > scan_limit then
            return nil, nil, ERR_MALFORMED
        end
        if byte(data, pos) == 125 then
            return item_id, pos + 1, nil
        end
        if byte(data, pos) ~= 34 then
            return nil, nil, ERR_MALFORMED
        end
        pos = pos + 1
        local key_start = pos
        while pos <= scan_limit and byte(data, pos) ~= 34 do
            pos = pos + 1
        end
        if pos > scan_limit then
            return nil, nil, ERR_MALFORMED
        end
        local key = sub(data, key_start, pos - 1)
        pos = pos + 1
        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if pos > scan_limit or byte(data, pos) ~= 58 then
            return nil, nil, ERR_MALFORMED
        end
        pos = pos + 1
        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if json_key_id(key) == "id" and item_id == nil then
            local raw
            raw, pos, err = read_json_string(data, pos, scan_limit)
            if not raw then
                return nil, nil, err
            end
            item_id = raw
        else
            local next_pos
            next_pos, err = skip_json_value(data, pos, scan_limit)
            if not next_pos then
                return nil, nil, err
            end
            pos = next_pos
        end
        while pos <= scan_limit and is_json_ws(byte(data, pos)) do
            pos = pos + 1
        end
        if pos > scan_limit then
            return nil, nil, ERR_MALFORMED
        end
        local sep = byte(data, pos)
        if sep == 125 then
            return item_id, pos + 1, nil
        end
        if sep ~= 44 then
            return nil, nil, ERR_MALFORMED
        end
        pos = pos + 1
    end
    return nil, nil, ERR_MALFORMED
end

local function scan_json_openrtb_dfa(data, scan_limit)
    local err
    local found = nil

    local function walk_object(pos)
        if byte(data, pos) ~= 123 then
            return nil, ERR_MALFORMED
        end
        pos = pos + 1
        while pos <= scan_limit do
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            if byte(data, pos) == 125 then
                return pos + 1, nil
            end
            if byte(data, pos) ~= 34 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
            local key_start = pos
            while pos <= scan_limit and byte(data, pos) ~= 34 do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            local key = sub(data, key_start, pos - 1)
            pos = pos + 1
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit or byte(data, pos) ~= 58 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end

            local kid = json_key_id(key)
            if kid == "item" and byte(data, pos) == 91 and found == nil then
                pos = pos + 1
                while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                    pos = pos + 1
                end
                if pos <= scan_limit and byte(data, pos) == 123 then
                    local id
                    id, pos, err = scan_item_object(data, pos, scan_limit)
                    if err then
                        return nil, err
                    end
                    found = id
                end
                local depth = 1
                while pos <= scan_limit and depth > 0 do
                    local c = byte(data, pos)
                    if c == 34 then
                        pos = pos + 1
                        while pos <= scan_limit and byte(data, pos) ~= 34 do
                            if byte(data, pos) == 92 then
                                pos = pos + 2
                            else
                                pos = pos + 1
                            end
                        end
                        pos = pos + 1
                    elseif c == 91 then
                        depth = depth + 1
                        pos = pos + 1
                    elseif c == 93 then
                        depth = depth - 1
                        pos = pos + 1
                    else
                        pos = pos + 1
                    end
                end
            elseif byte(data, pos) == 123 then
                local next_pos
                next_pos, err = walk_object(pos)
                if not next_pos then
                    return nil, err
                end
                pos = next_pos
            else
                local next_pos
                next_pos, err = skip_json_value(data, pos, scan_limit)
                if not next_pos then
                    return nil, err
                end
                pos = next_pos
            end

            while pos <= scan_limit and is_json_ws(byte(data, pos)) do
                pos = pos + 1
            end
            if pos > scan_limit then
                return nil, ERR_MALFORMED
            end
            local sep = byte(data, pos)
            if sep == 125 then
                return pos + 1, nil
            end
            if sep ~= 44 then
                return nil, ERR_MALFORMED
            end
            pos = pos + 1
        end
        return nil, ERR_MALFORMED
    end

    local pos = 1
    while pos <= scan_limit and is_json_ws(byte(data, pos)) do
        pos = pos + 1
    end
    local _, werr = walk_object(pos)
    if werr and not found then
        return nil, werr
    end
    return found, nil
end

function _M.check_content_length(content_length)
    if content_length and content_length > MAX_BODY_BYTES then
        return ERR_OVERSIZE
    end
    return nil
end

function _M.extract_campaign_id(body, content_length, schema)
    if content_length and content_length > MAX_BODY_BYTES then
        return nil, ERR_OVERSIZE
    end
    if not body or body == "" then
        return nil, nil
    end
    local body_len = #body
    if body_len > MAX_BODY_BYTES then
        return nil, ERR_OVERSIZE
    end

    local scan_limit = scan_limit_for(body_len, content_length)
    if scan_limit == 0 then
        return nil, nil
    end

    if not schema or schema == "" then
        schema = os.getenv "TRACKER_INGRESS_SCHEMA" or "ad_event_processor_native"
    end

    local pos = 1
    while pos <= scan_limit and is_json_ws(byte(body, pos)) do
        pos = pos + 1
    end
    local first = byte(body, pos)

    if first == 123 then
        if schema == "openrtb_3" then
            return scan_json_openrtb_dfa(body, scan_limit)
        end
        return scan_json_dfa(body, scan_limit)
    end
    if schema == "openrtb_3" then
        return nil, ERR_MALFORMED
    end
    return scan_proto_dfa(body, scan_limit)
end

_M.ERR_OVERSIZE = ERR_OVERSIZE
_M.ERR_MALFORMED = ERR_MALFORMED

return _M
