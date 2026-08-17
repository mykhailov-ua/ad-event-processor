-- OpenResty edge + Redis EVAL scripts; run: luacheck deploy/nginx/lua internal/ingestion
std = "max"
globals = {
    "ngx",
    "redis",
    "KEYS",
    "ARGV",
    "bit",
    "arg",
}
max_line_length = false
