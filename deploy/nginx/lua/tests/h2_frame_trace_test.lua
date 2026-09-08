#!/usr/bin/env lua
-- Unit tests for edge-h2-frame-trace.lua (delegates to edge-tcp-sig-v2).
local h2_frame_trace = require "edge-h2-frame-trace"

local passed = 0

local function assert_eq(got, want, label)
    if got ~= want then
        error(string.format("%s: got %s want %s", label, tostring(got), tostring(want)))
    end
    passed = passed + 1
end

local function assert_nil(got, label)
    if got ~= nil then
        error(string.format("%s: expected nil got %s", label, tostring(got)))
    end
    passed = passed + 1
end

assert_eq(
    h2_frame_trace.validate_trace "settings:0,headers:1,window_update:2",
    "settings:0,headers:1,window_update:2",
    "h2 trace"
)
assert_eq(h2_frame_trace.validate_trace "  SETTINGS:0 , HEADERS:1 ", "settings:0,headers:1", "trim and lower-case")
assert_nil(h2_frame_trace.validate_trace "", "empty trace")
assert_nil(h2_frame_trace.validate_trace(string.rep("a", 513)), "trace too long")

print(string.format("h2_frame_trace_test: %d passed", passed))
