-- OpenResty edge + Redis EVAL scripts; run: bash scripts/ci/lint/lua.sh
std = "max"
globals = {
    "ngx",
    "redis",
    "KEYS",
    "ARGV",
    "bit",
    "arg",
    "wrk",
    "request",
    "response",
    "setup",
    "thread",
    "done",
}
max_line_length = false
