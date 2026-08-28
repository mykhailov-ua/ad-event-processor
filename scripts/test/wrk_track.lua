local pct_openrtb_bid = tonumber(os.getenv("WRK_PCT_OPENRTB_BID") or "5")
local pct_click = tonumber(os.getenv("WRK_PCT_CLICK") or os.getenv("WRK_MIX_CLICK_PCT") or "20")
local track_body_mode = os.getenv("WRK_TRACK_BODY") or "openrtb3"
local counter = 0

local function use_native_track_body()
  return track_body_mode == "native" or track_body_mode == "ad_event_processor_native"
end

local function campaign_for(iter)
  local n = (iter % 100) + 1
  return string.format("00000000-0000-0000-0000-%012x", n)
end

local function native_track_body(iter, cid)
  return string.format(
    '{"campaign_id":"%s","user_id":"u-%d","type":"impression","click_id":"clk-%d","payload":{"slot":"top","cpm":1.25}}',
    cid,
    iter,
    iter
  )
end

local function openrtb3_track_body(iter, cid)
  return string.format(
    '{"openrtb":{"ver":"3.0","request":{"id":"req-%d","item":[{"id":"%s","flr":1.25,"spec":{"placement":{"tagid":"plc-wrk-%d"}}}],"context":{"device":{"type":2,"ip":"203.0.113.%d","ua":"Mozilla/5.0 (wrk loadgen)"},"site":{"page":"https://example.com/offer"}}}},"category_mask":4,"deal_id":"deal-wrk-%d"}',
    iter,
    cid,
    iter % 1000,
    (iter % 240) + 1,
    iter % 10000
  )
end

local function track_body(iter, cid)
  if use_native_track_body() then
    return native_track_body(iter, cid)
  end
  return openrtb3_track_body(iter, cid)
end

local function openrtb_bid_body(iter)
  return string.format(
    '{"id":"load-%d","tmax":300,"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":300,"h":250}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0 (wrk)","devicetype":2,"geo":{"country":"US"}}}',
    iter
  )
end

local function click_path(iter, cid)
  local click_id = string.format("00000000-0000-4000-8000-%012x", iter)
  return string.format(
    "/click?campaign_id=%s&click_id=%s&sub1=wrk&sub2=loadgen",
    cid,
    click_id
  )
end

local json_headers = {
  ["Content-Type"] = "application/json",
  ["Accept"] = "application/json",
  ["Connection"] = "keep-alive",
}

local bid_headers = {
  ["Content-Type"] = "application/json",
  ["Accept"] = "application/json",
  ["Connection"] = "keep-alive",
  ["x-openrtb-version"] = "2.6",
}

local click_headers = {
  ["Accept"] = "*/*",
  ["Connection"] = "keep-alive",
  ["User-Agent"] = "Mozilla/5.0 (wrk click)",
}

request = function()
  counter = counter + 1
  local roll = counter % 100
  local cid = campaign_for(counter)

  if roll < pct_openrtb_bid then
    return wrk.format("POST", "/openrtb/bid", bid_headers, openrtb_bid_body(counter))
  end

  if roll < (pct_openrtb_bid + pct_click) then
    return wrk.format("GET", click_path(counter, cid), click_headers)
  end

  return wrk.format("POST", "/track", json_headers, track_body(counter, cid))
end
