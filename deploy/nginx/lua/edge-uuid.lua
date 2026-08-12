
local _M = {}

-- 8-4-4-4-12 lowercase hex (canonical UUID string form).
local UUID_PATTERN = "^%x%x%x%x%x%x%x%x%-%x%x%x%x%-%x%x%x%x%-%x%x%x%x%-%x%x%x%x%x%x%x%x%x%x%x%x$"

function _M.normalize(s)
    if type(s) ~= "string" or #s ~= 36 then
        return nil
    end
    if string.find(s, "\0", 1, true) then
        return nil
    end
    if not string.match(s, UUID_PATTERN) then
        return nil
    end
    return string.lower(s)
end

function _M.to_bytes(normalized)
    if type(normalized) ~= "string" or #normalized ~= 36 then
        return nil
    end
    local hex = normalized:gsub("-", "")
    if #hex ~= 32 then
        return nil
    end
    local bytes = {}
    for i = 1, 32, 2 do
        local n = tonumber(hex:sub(i, i + 1), 16)
        if not n then
            return nil
        end
        bytes[#bytes + 1] = string.char(n)
    end
    return table.concat(bytes)
end

function _M.normalize_to_bytes(s)
    local normalized = _M.normalize(s)
    if not normalized then
        return nil
    end
    return _M.to_bytes(normalized)
end

return _M
