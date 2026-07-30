
package.path = arg[1] .. "/?.lua;;"

local WEIGHT_SCALE = 1000000
local STALE_EPOCH_LAG = 2
local SYNC_INTERVAL_SEC = 10

local dict_store = {}

ngx = {
    time = function()
        return os.time()
    end,
    now = function()
        return os.clock()
    end,
    log = {
        WARN = function() end,
    },
    shared = {
        node_weights = {
            get = function(_, key)
                return dict_store[key]
            end,
            set = function(_, key, val)
                dict_store[key] = val
            end,
        },
    },
    crc32_short = function(s)
        local h = 0
        for i = 1, #s do
            h = (h * 31 + string.byte(s, i)) % 2147483647
        end
        return h
    end,
    var = {
        request_id = "test-req",
    },
}

local node_weights = require("edge-node-weights")

local passed, failed = 0, 0

local function assert_true(cond, msg)
    if cond then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write("FAIL: " .. msg .. "\n")
    end
end

local function assert_near(want, got, eps, msg)
    if math.abs(want - got) <= eps then
        passed = passed + 1
    else
        failed = failed + 1
        io.stderr:write(string.format("FAIL: %s want=%.4f got=%.4f\n", msg, want, got))
    end
end

dict_store = {}
dict_store["peer_count"] = 2
dict_store["w:0"] = math.floor(0.25 * WEIGHT_SCALE + 0.5)
dict_store["w:1"] = math.floor(0.75 * WEIGHT_SCALE + 0.5)
dict_store["sync_ts"] = ngx.time()
dict_store["epoch_lag"] = 0
dict_store["sync_interval"] = SYNC_INTERVAL_SEC

local counts = { [0] = 0, [1] = 0 }
local trials = 10000
for i = 1, trials do
    ngx.var.request_id = "req-" .. i
    local idx = node_weights.pick_peer_index()
    counts[idx] = (counts[idx] or 0) + 1
end

local ratio0 = counts[0] / trials
local ratio1 = counts[1] / trials
assert_near(0.25, ratio0, 0.05, "weighted ratio peer 0")
assert_near(0.75, ratio1, 0.05, "weighted ratio peer 1")

dict_store["sync_ts"] = ngx.time() - (SYNC_INTERVAL_SEC * STALE_EPOCH_LAG + 1)
assert_true(node_weights.stale(), "stale when sync aged out")
assert_true(node_weights.drain_frozen(), "drain frozen when stale")

counts = { [0] = 0, [1] = 0 }
for i = 1, trials do
    ngx.var.request_id = "stale-" .. i
    local idx = node_weights.pick_peer_index()
    counts[idx] = (counts[idx] or 0) + 1
end
ratio0 = counts[0] / trials
ratio1 = counts[1] / trials
assert_near(0.5, ratio0, 0.05, "equalized ratio peer 0")
assert_near(0.5, ratio1, 0.05, "equalized ratio peer 1")

dict_store = {}
dict_store["peer_count"] = 2
dict_store["w:0"] = math.floor(0.25 * WEIGHT_SCALE + 0.5)
dict_store["w:1"] = math.floor(0.75 * WEIGHT_SCALE + 0.5)
dict_store["sync_ts"] = ngx.time()
dict_store["epoch_lag"] = 3
dict_store["sync_interval"] = SYNC_INTERVAL_SEC
assert_true(node_weights.stale(), "stale when epoch_lag > 2")

assert_true(not node_weights.fail_open(), "fail_open defaults off")
assert_true(node_weights.drain_frozen(), "conservative freezes drain when stale")

package.loaded["edge-node-weights"] = nil
local orig_getenv = os.getenv
local node_weights_fo = require("edge-node-weights")
node_weights_fo.set_getenv_for_test(function(name)
    if name == "CONTROL_FAIL_OPEN" then
        return "1"
    end
    return orig_getenv(name)
end)

dict_store = {}
dict_store["peer_count"] = 2
dict_store["w:0"] = math.floor(0.25 * WEIGHT_SCALE + 0.5)
dict_store["w:1"] = math.floor(0.75 * WEIGHT_SCALE + 0.5)
dict_store["sync_ts"] = ngx.time() - (SYNC_INTERVAL_SEC * STALE_EPOCH_LAG + 1)
dict_store["epoch_lag"] = 0
dict_store["sync_interval"] = SYNC_INTERVAL_SEC
assert_true(node_weights_fo.fail_open(), "fail_open env enabled")
assert_true(node_weights_fo.stale(), "still stale under fail-open")
assert_true(not node_weights_fo.drain_frozen(), "fail-open does not freeze drain")

counts = { [0] = 0, [1] = 0 }
for i = 1, trials do
    ngx.var.request_id = "fo-" .. i
    local idx = node_weights_fo.pick_peer_index()
    counts[idx] = (counts[idx] or 0) + 1
end
ratio0 = counts[0] / trials
ratio1 = counts[1] / trials
assert_near(0.25, ratio0, 0.05, "fail-open keeps 0.25 weight")
assert_near(0.75, ratio1, 0.05, "fail-open keeps 0.75 weight")

io.write(string.format("node_weights_test: passed=%d failed=%d\n", passed, failed))
if failed > 0 then
    os.exit(1)
end
