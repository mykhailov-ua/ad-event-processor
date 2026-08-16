
local _M = {}

function _M.is_unix_socket(addr)
    if not addr or addr == "" then
        return false
    end
    return addr:sub(1, 1) == "/" or string.find(addr, ".sock", 1, true) ~= nil
end

function _M.parse_redis_addr(addr)
    addr = addr:match("^%s*(.-)%s*$") or ""
    if addr == "" then
        return nil
    end
    if _M.is_unix_socket(addr) then
        return { unix_socket = addr }
    end
    local host, port = addr:match("([^:]+):(%d+)")
    if host and port then
        return { host = host, port = tonumber(port) }
    end
    return nil
end

function _M.parse_addr_list(raw)
    local out = {}
    if not raw or raw == "" then
        return out
    end
    for addr in string.gmatch(raw, "([^,]+)") do
        local parsed = _M.parse_redis_addr(addr)
        if parsed then
            table.insert(out, parsed)
        end
    end
    return out
end

function _M.redis_connect(red, target)
    if target.unix_socket then
        return red:connect("unix:", 0, { pool = "_", server = target.unix_socket })
    end
    return red:connect(target.host, target.port)
end

function _M.socket_connect(sock, target)
    if target.unix_socket then
        return sock:connect("unix:", 0, { pool = "_", server = target.unix_socket })
    end
    return sock:connect(target.host, target.port)
end

function _M.parse_http_url(url)
    if not url then
        return nil
    end
    if url:sub(1, 7) == "unix://" then
        local path = url:sub(8)
        local q = path:find("?", 1, true)
        if q then
            path = path:sub(1, q - 1)
        end
        return { unix_socket = path, path = "/" }
    end
    local host, port, path = string.match(url, "^https?://([^:/]+):?(%d*)(/.*)$")
    if not host then
        host, path = string.match(url, "^https?://([^/]+)(/.*)$")
        port = ""
    end
    if not host then
        return nil
    end
    if port == "" or port == nil then
        port = 8188
    else
        port = tonumber(port)
    end
    if not path or path == "" then
        path = "/"
    end
    return { host = host, port = port, path = path }
end

function _M.http_get_json(url)
    local parsed = _M.parse_http_url(url)
    if not parsed then
        return nil, "invalid url"
    end
    local sock = ngx.socket.tcp()
    sock:settimeout(2000)
    local ok, err = _M.socket_connect(sock, parsed)
    if not ok then
        return nil, err
    end
    local host_header = parsed.host or "localhost"
    local req = "GET " .. (parsed.path or "/") .. " HTTP/1.1\r\nHost: " .. host_header .. "\r\nConnection: close\r\nAccept: application/json\r\n\r\n"
    local sent, send_err = sock:send(req)
    if not sent then
        sock:close()
        return nil, send_err
    end
    local data, read_err = sock:receive("*a")
    sock:close()
    if not data then
        return nil, read_err
    end
    local body = string.match(data, "\r\n\r\n(.*)$")
    if not body then
        return nil, "empty http body"
    end
    local cjson = require "cjson.safe"
    return cjson.decode(body)
end

return _M
