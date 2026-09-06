-- Perimeter client IP canonicalization for generational blacklist keys b:{ip} and edge_rl ip: subjects.
-- Runtime: access-check.lua, edge-blacklist-sync stamp_ips, edge-rl rl_subject_key.
-- Uses libc inet_pton/inet_ntop under OpenResty luajit; maps IPv4-mapped IPv6 to dotted quad (Go net.ParseIP To4 parity).
--
-- Returns: canonical string; unparseable input returned unchanged (fail-open lookup miss, not 403).
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-ip.lua
-- bash scripts/test/edge/lua_tests.sh unit
local _M = {}

local ffi_ok, ffi = pcall(require, "ffi")
local AF_INET = 2
local AF_INET6 = 10
local INET_ADDRSTRLEN = 16
local INET6_ADDRSTRLEN = 46

local in4
local in6
local out_buf
local inet_pton
local inet_ntop

if ffi_ok then
    pcall(function()
        ffi.cdef [[
        typedef unsigned int socklen_t;
        struct in_addr { unsigned int s_addr; };
        struct in6_addr { unsigned char s6_addr[16]; };
        int inet_pton(int af, const char *src, void *dst);
        const char *inet_ntop(int af, const void *src, char *dst, socklen_t size);
    ]]
    end)
    in4 = ffi.new "struct in_addr"
    in6 = ffi.new "struct in6_addr"
    out_buf = ffi.new("char[?]", INET6_ADDRSTRLEN)
    inet_pton = ffi.C.inet_pton
    inet_ntop = ffi.C.inet_ntop
end

local function ntop4()
    local p = inet_ntop(AF_INET, in4, out_buf, INET_ADDRSTRLEN)
    if p == nil then
        return nil
    end
    return ffi.string(p)
end

local function is_v4mapped(in6_addr)
    for i = 0, 9 do
        if in6_addr.s6_addr[i] ~= 0 then
            return false
        end
    end
    return in6_addr.s6_addr[10] == 0xff and in6_addr.s6_addr[11] == 0xff
end

local function canonical_fallback(ip)
    local lower = ip:lower()
    local v4 = lower:match "^::ffff:(%d+%.%d+%.%d+%.%d+)$"
    if v4 then
        return v4
    end
    return ip
end

function _M.canonical(ip)
    if type(ip) ~= "string" or ip == "" then
        return ip
    end

    if not ffi_ok then
        return canonical_fallback(ip)
    end

    if inet_pton(AF_INET, ip, in4) == 1 then
        local s = ntop4()
        if s then
            return s
        end
    end

    if inet_pton(AF_INET6, ip, in6) == 1 then
        if is_v4mapped(in6) then
            ffi.copy(in4, in6.s6_addr + 12, 4)
            local s = ntop4()
            if s then
                return s
            end
        end
        local p = inet_ntop(AF_INET6, in6, out_buf, INET6_ADDRSTRLEN)
        if p ~= nil then
            return ffi.string(p)
        end
    end

    return ip
end

return _M
