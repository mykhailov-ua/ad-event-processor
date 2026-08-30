-- Safe-page response substitution when tracker sets X-Ad-Event-Processor-Safe-Page header.
-- Runtime: nginx worker header_filter + body_filter phases on proxy response.
--
-- Topology: internal location /safe_page_content fetches HTML; campaign_id from ngx.ctx.
--
-- HTTP: replaces upstream body with safe page HTML or minimal fallback; strips internal header.
--
-- Forbidden: tarpit or safe-page on settlement/billing upstream paths.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-safe-page.lua
-- bash scripts/test/edge/lua_tests.sh
local _M = {}

function _M.header_filter()
    if ngx.header["X-Ad-Event-Processor-Safe-Page"] then
        ngx.header["X-Ad-Event-Processor-Safe-Page"] = nil
        ngx.ctx.safe_page = true
        ngx.header.content_length = nil
    end
end

function _M.body_filter()
    if not ngx.ctx.safe_page then
        return
    end
    if ngx.arg[2] then
        if not ngx.ctx.safe_body then
            local cid = ngx.ctx.campaign_id or ""
            local res = ngx.location.capture("/safe_page_content", {
                method = ngx.HTTP_GET,
                body = "",
                ctx = {},
                vars = {},
                args = { campaign_id = cid },
                copy_all_vars = false,
                share_all_vars = false,
                always_forward_body = false,
            })
            if res.status == 200 and res.body and res.body ~= "" then
                ngx.ctx.safe_body = res.body
            else
                ngx.ctx.safe_body = "<!DOCTYPE html><html><body><p>Unavailable</p></body></html>"
            end
        end
        ngx.arg[1] = ngx.ctx.safe_body
    else
        ngx.arg[1] = nil
    end
end

return _M
