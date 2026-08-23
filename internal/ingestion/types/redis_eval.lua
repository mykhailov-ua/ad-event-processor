---@meta
-- Redis EVAL globals injected at script runtime.

---@class redis
---@field call fun(cmd: string, ...): any
redis = {}

---@type string[]
KEYS = {}

---@type string[]
ARGV = {}
