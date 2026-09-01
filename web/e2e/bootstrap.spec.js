import { test, expect } from '@playwright/test';

import { skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('GET / serves SPA shell with root mount', async ({ request }) => {
  const response = await request.get('/');
  expect(response.ok()).toBeTruthy();

  const html = await response.text();
  expect(html).toMatch(/<div id="root"/);
});
