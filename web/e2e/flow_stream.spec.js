import { test, expect } from '@playwright/test';

import {
  baseURL,
  integrationRunToken,
  loginAsAdmin,
  probeTrackerBaseUrl,
  skipUnlessIntegrationReady,
  trackerBaseURL,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('POST /api/v1/flows/validate returns 400 when path weights do not sum to 100', async ({
  page,
}) => {
  await loginAsAdmin(page);

  const landerResponse = await page.request.post(new URL('/api/v1/landers', baseURL).toString(), {
    data: { name: `e2e-lander-${integrationRunToken()}`, url: 'https://e2e-lander.example/lp' },
  });
  expect(landerResponse.ok()).toBeTruthy();
  const lander = await landerResponse.json();

  const offerResponse = await page.request.post(new URL('/api/v1/offers', baseURL).toString(), {
    data: { name: `e2e-offer-${integrationRunToken()}`, url: 'https://e2e-offer.example' },
  });
  expect(offerResponse.ok()).toBeTruthy();
  const offer = await offerResponse.json();

  const validateResponse = await page.request.post(
    new URL('/api/v1/flows/validate', baseURL).toString(),
    {
      data: {
        paths: [
          {
            weight: 50,
            landers: [{ lander_id: lander.id, weight: 100 }],
            offers: [{ offer_id: offer.id, weight: 100 }],
          },
          {
            weight: 49,
            landers: [{ lander_id: lander.id, weight: 100 }],
            offers: [{ offer_id: offer.id, weight: 100 }],
          },
        ],
      },
    }
  );
  expect(validateResponse.status()).toBe(400);
  const body = await validateResponse.json();
  expect(body.valid).toBe(false);
  expect(body.path_errors?.[0]?.code).toBe('weight_sum');
});

test('create flow dialog shows inline validation when weights do not sum to 100', async ({
  page,
}) => {
  await loginAsAdmin(page);
  await page.goto('/flows');
  await page.getByRole('button', { name: 'Create flow' }).click();
  await page.getByLabel('Weight %').first().fill('50');
  await expect(page.getByText('Flow validation')).toBeVisible();
  await expect(page.getByText(/sum to 100/i)).toBeVisible();
});

test(
  'flow attached to campaign redirects /click to flow lander URL',
  { tag: '@write' },
  async ({ page }, testInfo) => {
    const trackerReady = await probeTrackerBaseUrl();
    if (!trackerReady) {
      testInfo.skip(true, `integration: tracker unreachable at ${trackerBaseURL}`);
      return;
    }

    await loginAsAdmin(page);

    const campaignsResponse = await page.request.get(
      new URL('/api/v1/campaigns?limit=1&status=ACTIVE', baseURL).toString()
    );
    expect(campaignsResponse.ok()).toBeTruthy();
    const campaignsBody = await campaignsResponse.json();
    const campaign = campaignsBody?.items?.[0];
    if (!campaign?.id) {
      testInfo.skip(true, 'integration: no active campaigns for flow click routing');
      return;
    }

    const runToken = integrationRunToken();
    const landerHost = `e2e-flow-${runToken}.invalid`;
    const landerURL = `https://${landerHost}/lp`;

    const landerResponse = await page.request.post(new URL('/api/v1/landers', baseURL).toString(), {
      data: { name: `e2e-flow-lander-${runToken}`, url: landerURL },
    });
    expect(landerResponse.ok()).toBeTruthy();
    const lander = await landerResponse.json();

    const offerResponse = await page.request.post(new URL('/api/v1/offers', baseURL).toString(), {
      data: { name: `e2e-flow-offer-${runToken}`, url: 'https://e2e-offer.example' },
    });
    expect(offerResponse.ok()).toBeTruthy();
    const offer = await offerResponse.json();

    const flowResponse = await page.request.post(new URL('/api/v1/flows', baseURL).toString(), {
      data: {
        name: `e2e-flow-${runToken}`,
        paths: [
          {
            weight: 100,
            landers: [{ lander_id: lander.id, weight: 100 }],
            offers: [{ offer_id: offer.id, weight: 100 }],
          },
        ],
      },
    });
    expect(flowResponse.ok()).toBeTruthy();
    const flow = await flowResponse.json();

    const patchResponse = await page.request.patch(
      new URL(`/api/v1/campaigns/${campaign.id}`, baseURL).toString(),
      {
        data: { flow_id: flow.id },
      }
    );
    expect(patchResponse.ok()).toBeTruthy();

    const clickURL = new URL('/click', trackerBaseURL);
    clickURL.searchParams.set('campaign_id', campaign.id);
    clickURL.searchParams.set('type', 'click');
    clickURL.searchParams.set('click_id', `e2e-${runToken}`);

    let location = '';
    for (let attempt = 0; attempt < 8; attempt += 1) {
      const clickResponse = await page.request.get(clickURL.toString(), {
        maxRedirects: 0,
      });
      if (clickResponse.status() === 302 || clickResponse.status() === 303) {
        location = clickResponse.headers()['location'] ?? '';
        if (location.includes(landerHost)) {
          break;
        }
      }
      await page.waitForTimeout(500);
    }

    expect(location).toContain(landerHost);
  }
);
