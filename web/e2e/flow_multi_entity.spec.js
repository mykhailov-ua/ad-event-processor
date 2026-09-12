import { test, expect } from '@playwright/test';

import {
  apiMutationHeaders,
  baseURL,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('PUT flow persists second offer on path', { tag: '@write' }, async ({ page }) => {
  await loginAsAdmin(page);

  const flowsResponse = await page.request.get(new URL('/api/v1/flows', baseURL).toString());
  if (!flowsResponse.ok()) {
    test.skip(true, `integration: flows list ${flowsResponse.status()}`);
    return;
  }
  const flows = await flowsResponse.json();
  const flow = Array.isArray(flows) ? flows[0] : flows.items?.[0];
  if (!flow?.id) {
    test.skip(true, 'integration: no flows available');
    return;
  }

  const offersResponse = await page.request.get(
    new URL('/api/v1/offers?limit=50', baseURL).toString()
  );
  expect(offersResponse.ok()).toBeTruthy();
  const offersBody = await offersResponse.json();
  const offers = Array.isArray(offersBody) ? offersBody : (offersBody.items ?? []);
  if (offers.length < 2) {
    test.skip(true, 'integration: need at least two offers for multi-entity path');
    return;
  }

  const landersResponse = await page.request.get(
    new URL('/api/v1/landers?limit=50', baseURL).toString()
  );
  expect(landersResponse.ok()).toBeTruthy();
  const landersBody = await landersResponse.json();
  const landers = Array.isArray(landersBody) ? landersBody : (landersBody.items ?? []);
  if (landers.length < 1) {
    test.skip(true, 'integration: need at least one lander for flow path');
    return;
  }

  const getFlowResponse = await page.request.get(
    new URL(`/api/v1/flows/${flow.id}`, baseURL).toString()
  );
  expect(getFlowResponse.ok()).toBeTruthy();
  const existing = await getFlowResponse.json();
  const offerA = offers[0];
  const offerB = offers[1];
  const lander = landers[0];

  const paths = [
    {
      weight: 100,
      rotation_mode: 'unseen',
      landers: [{ lander_id: lander.id, weight: 100 }],
      offers: [
        { offer_id: offerA.id, weight: 60 },
        { offer_id: offerB.id, weight: 40 },
      ],
    },
  ];

  const putResponse = await page.request.put(
    new URL(`/api/v1/flows/${flow.id}`, baseURL).toString(),
    {
      data: {
        name: existing.name,
        paths,
        flow_routing_mode: existing.flow_routing_mode ?? 'weighted',
      },
      headers: await apiMutationHeaders(page),
    }
  );
  expect(putResponse.ok()).toBeTruthy();

  const verifyResponse = await page.request.get(
    new URL(`/api/v1/flows/${flow.id}`, baseURL).toString()
  );
  expect(verifyResponse.ok()).toBeTruthy();
  const saved = await verifyResponse.json();
  const savedPaths = Array.isArray(saved.paths) ? saved.paths : JSON.parse(saved.paths);
  expect(savedPaths[0].offers).toHaveLength(2);
  expect(savedPaths[0].offers.map((row) => row.offer_id)).toEqual(
    expect.arrayContaining([offerA.id, offerB.id])
  );
  expect(savedPaths[0].rotation_mode).toBe('unseen');
});
