local _M = {}

local EXT_SUPPORTED_GROUPS = 10
local EXT_EC_POINT_FORMATS = 11
local EXT_ALPN = 16

local VERSION_MAP = {
    ["TLSv1.3"] = 772,
    ["TLSv1.2"] = 771,
    ["TLSv1.1"] = 770,
    ["TLSv1"] = 769,
    ["SSLv3"] = 768,
}

local function join_nums(nums, sep)
    if not nums or #nums == 0 then
        return ""
    end
    return table.concat(nums, sep)
end

local function parse_u16_be_list(ext_bytes)
    if not ext_bytes or #ext_bytes < 2 then
        return {}
    end
    local n = ext_bytes:byte(1) * 256 + ext_bytes:byte(2)
    local out = {}
    local off = 3
    local limit = math.min(n / 2, (#ext_bytes - 2) / 2)
    for _ = 1, limit do
        if off + 1 > #ext_bytes then
            break
        end
        local v = ext_bytes:byte(off) * 256 + ext_bytes:byte(off + 1)
        out[#out + 1] = v
        off = off + 2
    end
    return out
end

local function parse_u8_list(ext_bytes)
    if not ext_bytes or #ext_bytes < 1 then
        return {}
    end
    local n = ext_bytes:byte(1)
    local out = {}
    for i = 1, n do
        if i + 1 <= #ext_bytes then
            out[#out + 1] = ext_bytes:byte(i + 1)
        end
    end
    return out
end

local function parse_alpn_list(ext_bytes)
    if not ext_bytes or #ext_bytes < 2 then
        return ""
    end
    local list_len = ext_bytes:byte(1) * 256 + ext_bytes:byte(2)
    local off = 3
    local protos = {}
    local consumed = 0
    while consumed < list_len and off <= #ext_bytes do
        local plen = ext_bytes:byte(off)
        if not plen or plen == 0 or off + plen > #ext_bytes then
            break
        end
        protos[#protos + 1] = ext_bytes:sub(off + 1, off + plen)
        off = off + 1 + plen
        consumed = consumed + 1 + plen
    end
    return table.concat(protos, ",")
end

local function ja3_version(ssl_clt)
    local vers = ssl_clt.get_supported_versions()
    if vers and #vers > 0 then
        local best = 771
        for _, name in ipairs(vers) do
            local n = VERSION_MAP[name]
            if n and n > best then
                best = n
            end
        end
        return best
    end
    return 771
end

function _M.build_ja3_from_parts(version, ciphers, extensions, curves, ec_fmt)
    return string.format(
        "%d,%s,%s,%s,%s",
        version,
        join_nums(ciphers, "-"),
        join_nums(extensions, "-"),
        join_nums(curves, "-"),
        join_nums(ec_fmt, "-")
    )
end

function _M.build_ja4_from_parts(version, sni_present, ciphers, extensions)
    local cipher_str = join_nums(ciphers, ",")
    local ext_str = join_nums(extensions, ",")
    local vlabel = version == 772 and "13" or "12"
    local sni_flag = sni_present and "d" or "i"
    return string.format(
        "t%s%s%02d%02d_%s_%s",
        vlabel,
        sni_flag,
        #ciphers,
        #extensions,
        ngx.md5(cipher_str):sub(1, 12),
        ngx.md5(ext_str):sub(1, 12)
    )
end

function _M.compute(ctx)
    if not ctx then
        return
    end

    local ok, ssl_clt = pcall(require, "ngx.ssl.clienthello")
    if not ok then
        return
    end

    local ciphers = ssl_clt.get_client_hello_ciphers()
    if not ciphers then
        return
    end

    local extensions = ssl_clt.get_client_hello_ext_present() or {}
    table.sort(extensions)

    local version = ja3_version(ssl_clt)
    local curves = parse_u16_be_list(ssl_clt.get_client_hello_ext(EXT_SUPPORTED_GROUPS))
    local ec_fmt = parse_u8_list(ssl_clt.get_client_hello_ext(EXT_EC_POINT_FORMATS))

    ctx.tls_ja3 = _M.build_ja3_from_parts(version, ciphers, extensions, curves, ec_fmt)

    local sni = ssl_clt.get_client_hello_server_name()
    ctx.tls_ja4 = _M.build_ja4_from_parts(version, sni and sni ~= "", ciphers, extensions)

    local alpn_ext = ssl_clt.get_client_hello_ext(EXT_ALPN)
    if alpn_ext and alpn_ext ~= "" then
        ctx.tls_alpn = parse_alpn_list(alpn_ext)
    end
end

_M.parse_alpn_list = parse_alpn_list

return _M
