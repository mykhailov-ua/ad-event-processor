package.path = arg[1] .. "/?.lua;;"
local fp = require "edge-tls-fingerprint"

local passed, failed = 0, 0

local function assert_case(name, fn)
    local ok, err = pcall(fn)
    if ok then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL %s: %s\n", name, tostring(err)))
    end
end

assert_case("build_ja3_from_parts", function()
    local ja3 = fp.build_ja3_from_parts(771, { 4865, 4866 }, { 0, 23, 29 }, { 29, 23, 24 }, { 0 })
    if ja3 ~= "771,4865-4866,0-23-29,29-23-24,0" then
        error("unexpected ja3: " .. ja3)
    end
end)

assert_case("build_ja4_from_parts_format", function()
    if not ngx then
        rawset(_G, "ngx", {
            md5 = function(_)
                return string.rep("a", 32)
            end,
        })
    end
    local ja4 = fp.build_ja4_from_parts(772, true, { 4865 }, { 0, 23 })
    if not ja4:match "^t13d0102_" then
        error("unexpected ja4 prefix: " .. ja4)
    end
end)

print(string.format("edge_tls_fingerprint: passed=%d failed=%d", passed, failed))
if failed > 0 then
    os.exit(1)
end
