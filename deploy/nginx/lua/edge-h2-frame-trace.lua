-- Validate X-H2-FRAME-TRACE before SHM cache or upstream header forward.
-- Runtime: edge-tcp-fp-sync worker timer and edge-ingress access phase (read-only).
-- Parity: pkg/h2frametrace NormalizeTokens (512-byte trace cap, 64-byte token cap, lower-case ASCII).
--
-- Returns normalized comma-separated trace or nil when invalid (fail-open: omit header).
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-h2-frame-trace.lua
-- bash scripts/test/edge/lua_tests.sh
local tcp_sig_v2 = require "edge-tcp-sig-v2"

local _M = {}

function _M.validate_trace(raw)
    return tcp_sig_v2.validate_trace(raw)
end

return _M
