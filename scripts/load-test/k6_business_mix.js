// k6 business-mix load test: 50% broken / 20% gray (hard to detect) / 30% clean.
//
// Broken: malformed payloads, unknown campaigns, protocol abuse, edge noise.
// Gray: valid shape + seeded campaign but fraud/geo/datacenter signals or dedup storms.
// Clean: valid JSON, seeded campaign IDs, unique click_id, no abuse headers.
//
// Env: RATE, DURATION, TRACKER_BASES, EDGE_URL, PREALLOC_VUS, MAX_VUS, OVERSIZE_BYTES
//      PCT_BROKEN (50), PCT_GRAY (20) — clean = remainder

import http from 'k6/http';
import { check } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import exec from 'k6/execution';

const rate = parseInt(__ENV.RATE || '2000', 10);
const duration = __ENV.DURATION || '2m';
const edgeURL = __ENV.EDGE_URL || '';
const oversizeBytes = parseInt(__ENV.OVERSIZE_BYTES || '65536', 10);
const preAllocVUs = parseInt(__ENV.PREALLOC_VUS || '200', 10);
const maxVUs = parseInt(__ENV.MAX_VUS || '800', 10);
const pctBroken = parseFloat(__ENV.PCT_BROKEN || '50');
const pctGray = parseFloat(__ENV.PCT_GRAY || '20');
const pctClean = 100 - pctBroken - pctGray;

const trackerBases = (__ENV.TRACKER_BASES || 'http://127.0.0.1:8181,http://127.0.0.1:8182')
  .split(',')
  .map((s) => s.trim())
  .filter(Boolean);

const trafficBroken = new Counter('traffic_broken');
const trafficGray = new Counter('traffic_gray');
const trafficClean = new Counter('traffic_clean');
const acceptRate = new Rate('accepted_2xx');
const serverErrorRate = new Rate('server_5xx');
const clientErrorRate = new Rate('client_4xx');
const trackLatency = new Trend('track_latency_ms', true);

export const options = {
  scenarios: {
    business_mix: {
      executor: 'constant-arrival-rate',
      rate: rate,
      timeUnit: '1s',
      duration: duration,
      preAllocatedVUs: preAllocVUs,
      maxVUs: maxVUs,
    },
  },
  discardResponseBodies: true,
  thresholds: {
    track_latency_ms: ['p(99)<300'],
    http_req_duration: ['p(99)<500'],
  },
  tags: { test: 'business_mix' },
};

function pickTracker() {
  const idx = exec.scenario.iterationInTest % trackerBases.length;
  return trackerBases[idx];
}

function campaignID(vu, iter) {
  const n = ((vu * 997 + iter) % 100) + 1;
  return `00000000-0000-0000-0000-${n.toString(16).padStart(12, '0')}`;
}

function validBody(vu, iter, opts = {}) {
  const userId = opts.userId || `u-${vu}-${iter}`;
  const clickId = opts.clickId || `clk-${vu}-${iter}-${__ITER}`;
  return JSON.stringify({
    campaign_id: campaignID(vu, iter),
    user_id: userId,
    type: iter % 3 === 0 ? 'click' : 'impression',
    click_id: clickId,
    payload: { slot: 'top', cpm: 1.25 },
  });
}

function grayBody(vu, iter) {
  return JSON.stringify({
    campaign_id: campaignID(vu, iter),
    user_id: `gray-${vu}-bot`,
    type: 'click',
    click_id: `gclk-${vu}-${iter}`,
    payload: { bot: true, src: 'affiliate' },
  });
}

function fraudIP(iter) {
  return iter % 2 === 0 ? '203.0.113.66' : '198.51.100.77';
}

function classify(res, recordLatency) {
  if (recordLatency) trackLatency.add(res.timings.duration);
  if (res.status >= 200 && res.status < 300) acceptRate.add(1);
  else acceptRate.add(0);
  if (res.status >= 400 && res.status < 500) clientErrorRate.add(1);
  else clientErrorRate.add(0);
  if (res.status >= 500) serverErrorRate.add(1);
  else serverErrorRate.add(0);
}

function sendBroken(base, vu, iter, params) {
  const kind = iter % 8;
  let res;
  if (kind === 0) {
    res = http.post(`${base}/track`, '{not-json', {
      ...params,
      headers: { ...params.headers, 'Content-Type': 'application/json' },
    });
  } else if (kind === 1) {
    res = http.post(`${base}/track`, '\xff\xee\xdd\xcc\xbb', {
      ...params,
      headers: { ...params.headers, 'Content-Type': 'application/x-protobuf' },
    });
  } else if (kind === 2) {
    res = http.post(
      `${base}/track`,
      JSON.stringify({
        campaign_id: 'ffffffff-ffff-ffff-ffff-ffffffffffff',
        user_id: 'ghost',
        type: 'impression',
        click_id: `bad-${iter}`,
      }),
      { ...params, headers: { ...params.headers, 'Content-Type': 'application/json' } },
    );
  } else if (kind === 3) {
    res = http.post(`${base}/track`, 'x'.repeat(oversizeBytes), {
      ...params,
      headers: { ...params.headers, 'Content-Type': 'application/json' },
    });
  } else if (kind === 4) {
    res = http.get(`${base}/track`, params);
  } else if (kind === 5) {
    res = http.get(`${base}/health`, params);
  } else if (kind === 6) {
    res = http.post(`${base}/track`, null, {
      ...params,
      headers: { ...params.headers, 'Content-Type': 'application/json' },
    });
  } else if (edgeURL && kind === 7) {
    res = http.post(`${edgeURL}/track`, validBody(vu, iter), {
      ...params,
      headers: { ...params.headers, 'Content-Type': 'application/json' },
    });
  } else {
    res = http.post(`${base}/admin/boom`, '{}', {
      ...params,
      headers: { ...params.headers, 'Content-Type': 'application/json' },
    });
  }
  classify(res, false);
}

function sendGray(base, vu, iter, params) {
  const kind = iter % 4;
  let res;
  if (kind === 0) {
    res = http.post(`${base}/track`, grayBody(vu, iter), {
      ...params,
      headers: {
        ...params.headers,
        'Content-Type': 'application/json',
        'X-Forwarded-For': fraudIP(iter),
      },
    });
  } else if (kind === 1) {
    res = http.post(`${base}/track`, validBody(vu, iter, { userId: `fraud-bot-${vu}` }), {
      ...params,
      headers: {
        ...params.headers,
        'Content-Type': 'application/json',
        'X-Forwarded-For': fraudIP(iter),
      },
    });
  } else if (kind === 2) {
    const dup = JSON.stringify({
      campaign_id: campaignID(1, 1),
      user_id: 'gray-dup',
      type: 'click',
      click_id: 'gray-fixed-dedup-id',
      payload: {},
    });
    res = http.post(`${base}/track`, dup, {
      ...params,
      headers: { ...params.headers, 'Content-Type': 'application/json' },
    });
  } else {
    res = http.post(`${base}/track`, validBody(vu, iter), {
      ...params,
      headers: {
        ...params.headers,
        'Content-Type': 'application/json',
        'X-Forwarded-For': fraudIP(iter),
      },
    });
  }
  classify(res, true);
}

function sendClean(base, vu, iter, params) {
  const res = http.post(`${base}/track`, validBody(vu, iter), {
    ...params,
    headers: { ...params.headers, 'Content-Type': 'application/json' },
  });
  classify(res, true);
}

export default function () {
  const vu = __VU;
  const iter = __ITER;
  const roll = Math.random() * 100;
  const base = pickTracker();
  const params = { headers: { Connection: 'keep-alive' }, timeout: '10s' };

  if (roll < pctBroken) {
    trafficBroken.add(1);
    sendBroken(base, vu, iter, params);
    return;
  }
  if (roll < pctBroken + pctGray) {
    trafficGray.add(1);
    sendGray(base, vu, iter, params);
    return;
  }
  trafficClean.add(1);
  sendClean(base, vu, iter, params);
}

export function setup() {
  for (const base of trackerBases) {
    const res = http.get(`${base}/health`);
    check(res, { 'tracker healthy': (r) => r.status === 200 });
  }
  return {
    started: new Date().toISOString(),
    rate,
    duration,
    mix: { broken: pctBroken, gray: pctGray, clean: pctClean },
    trackers: trackerBases,
  };
}

export function teardown(data) {
  console.log(
    `business_mix done: rate=${data.rate} duration=${data.duration} ` +
      `mix=${data.mix.broken}/${data.mix.gray}/${data.mix.clean} trackers=${data.trackers.join(',')}`,
  );
}
