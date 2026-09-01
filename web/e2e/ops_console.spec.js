import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

const OPS_SECTION_PATHS = [
  '/ops/dlq',
  '/ops/blacklist',
  '/ops/incidents',
  '/ops/outbox',
  '/ops/shards',
  '/ops/ml-model',
  '/ops/domains',
  '/ops/recon',
  '/ops/consent',
  '/ops/rum',
  '/ops/metrics',
];

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('ops home shows section navigation', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/ops');
  await expect(page.getByRole('heading', { name: 'Ops' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Ops sections' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Incidents' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Metrics' })).toBeVisible();
});

test('ops metrics page shows live toggle', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/ops/metrics');
  await expect(page.getByRole('heading', { name: 'Dashboard metrics' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Live', exact: true })).toBeVisible();
});

test('ops home shows reload roles control', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/ops');
  await expect(page.getByRole('button', { name: 'Reload roles' })).toBeVisible();
});

test('ops section links resolve without 404', async ({ page }) => {
  await loginAsAdmin(page);

  for (const path of OPS_SECTION_PATHS) {
    await page.goto(path);
    await expect(page.getByRole('navigation', { name: 'Ops sections' })).toBeVisible();
    await expect(page.getByText('404')).toHaveCount(0);
    await expect(page.getByText('Page not found')).toHaveCount(0);
  }
});
