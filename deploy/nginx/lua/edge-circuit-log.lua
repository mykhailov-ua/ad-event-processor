-- log_by_lua hook: record upstream tracker failures for edge circuit breaker.
-- Runtime: nginx log phase on tracker proxy locations only.
--
-- Verify:
-- luac -p deploy/nginx/lua/edge-circuit-log.lua
local edge_circuit = require "edge-circuit"

edge_circuit.log_upstream_err()
