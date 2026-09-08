-- Validate X-TCP-SIG-V2 option-order trace before SHM cache or upstream header forward.
-- Runtime: edge-tcp-fp-sync worker timer and edge-ingress access phase (read-only).
-- Parity: pkg/tcpsynopt NormalizeTokens (512-byte trace cap, 64-byte token cap, lower-case ASCII).
--
-- Returns normalized comma-separated trace or nil when invalid (fail-open: omit header).
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-tcp-sig-v2.lua
-- bash scripts/test/edge/lua_tests.sh unit
local _M = {}

local MAX_TRACE_LEN = 512
local MAX_TOKEN_LEN = 64

local function trim(s)
    return (s:gsub("^%s+", ""):gsub("%s+$", ""))
end

function _M.validate_trace(raw)
    if type(raw) ~= "string" then
        return nil
    end
    raw = trim(raw)
    if raw == "" or #raw > MAX_TRACE_LEN then
        return nil
    end
    if raw:find("[%c]") then
        return nil
    end
    raw = string.lower(raw)
    local parts = {}
    for part in string.gmatch(raw, "[^,]+") do
        part = trim(part)
        if part ~= "" then
            if #part > MAX_TOKEN_LEN then
                return nil
            end
            parts[#parts + 1] = part
        end
    end
    if #parts == 0 then
        return nil
    end
    return table.concat(parts, ",")
end

return _M
