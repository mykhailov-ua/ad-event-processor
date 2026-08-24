local edge_config = require "edge-config"

local _M = {}

local function truthy(v)
    if v == nil then
        return false
    end
    if type(v) == "boolean" then
        return v
    end
    if type(v) == "number" then
        return v ~= 0
    end
    local s = string.lower(tostring(v))
    return s == "1" or s == "true" or s == "yes"
end

local function env_enabled(name)
    return truthy(os.getenv(name))
end

local function redis_enabled(field, env_name)
    local v = edge_config.get_flag(field)
    if v ~= nil then
        return truthy(v)
    end
    return env_enabled(env_name)
end

function _M.click_enabled()
    return redis_enabled("edge_expose_click", "EDGE_EXPOSE_CLICK")
end

function _M.openrtb_enabled()
    return redis_enabled("edge_expose_openrtb", "EDGE_EXPOSE_OPENRTB")
end

function _M.require_click()
    if not _M.click_enabled() then
        ngx.exit(ngx.HTTP_NOT_FOUND)
    end
end

function _M.require_openrtb()
    if not _M.openrtb_enabled() then
        ngx.exit(ngx.HTTP_NOT_FOUND)
    end
end

return _M
