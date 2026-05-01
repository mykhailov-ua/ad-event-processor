import http from 'k6/http';
import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

/**
 * AD-EVENT-PROCESSOR EXTREME STRESS TEST
 * 
 * Target: 10k - 100k Requests Per Second
 * Focus: Ingestion pipeline, Middleware performance, and Database batching logic.
 */

export const options = {
  scenarios: {
    // Phase 1: High Ingestion Baseline (20k RPS)
    high_load: {
      executor: 'constant-arrival-rate',
      rate: 20000,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 200,
      maxVUs: 1000,
    },
    // Phase 2: Flash Sale / Peak Stress (Ramping to 100k RPS)
    peak_stress: {
      executor: 'ramping-arrival-rate',
      startRate: 20000,
      timeUnit: '1s',
      stages: [
        { target: 50000, duration: '1m' },   // Ramp up to 50k
        { target: 100000, duration: '30s' }, // Extreme burst 100k
        { target: 0, duration: '30s' },       // Graceful ramp down
      ],
      preAllocatedVUs: 500,
      maxVUs: 3000,
      startTime: '1m',
    },
  },
  thresholds: {
    // 95% of requests should stay below 100ms even under extreme pressure.
    'http_req_duration': ['p(95)<100'],
    // Ensure failure rate (system level) is below 1%.
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://app:8085';
const CAMPAIGN_ID = '00000000-0000-0000-0000-000000000001';

export default function () {
  const url = `${BASE_URL}/track`;
  const rand = Math.random();
  
  let payload;
  let clickId = uuidv4();

  // Mixed traffic scenario to validate intelligent filtering performance
  if (rand < 0.6) {
    // 60% Clean traffic (unique click_ids)
    payload = JSON.stringify({
      campaign_id: CAMPAIGN_ID,
      type: 'impression',
      click_id: clickId
    });
  } else if (rand < 0.8) {
    // 20% Duplicate clicks (targets DuplicateEventFilter)
    payload = JSON.stringify({
      campaign_id: CAMPAIGN_ID,
      type: 'click',
      click_id: 'stress-test-global-duplicate-id'
    });
  } else {
    // 20% Aggressive IP bombardment (targets IPRateLimiter)
    payload = JSON.stringify({
      campaign_id: CAMPAIGN_ID,
      type: 'conversion',
      click_id: uuidv4()
    });
  }

  const params = { 
    headers: { 'Content-Type': 'application/json' },
    timeout: '5s'
  };
  
  http.post(url, payload, params);
}
