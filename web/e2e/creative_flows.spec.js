import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('flows visual create dialog exposes stream editor', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/flows');
  await page.getByRole('button', { name: 'Create flow' }).click();
  await expect(page.getByRole('heading', { name: 'Create flow' })).toBeVisible();
  await expect(page.getByRole('button', { name: '50 / 50 split' })).toBeVisible();
  await expect(page.getByLabel('Weight %')).toBeVisible();
});

test('flows list loads from GET /api/v1/flows', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/flows'), { timeout: 20_000 });
  await gotoLive(page, '/flows');
  await expect(page.getByRole('heading', { name: 'Flows' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Creative sections' })).toBeVisible();

  const response = await listResponse;
  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No flows',
    rowLabel: (row) => String(row.name ?? ''),
  });
  await expect(page.getByRole('button', { name: 'Create flow' })).toBeVisible();
});

test('landers list loads from GET /api/v1/landers', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/landers'), { timeout: 20_000 });
  await gotoLive(page, '/landers');
  await expect(page.getByRole('heading', { name: 'Landers' })).toBeVisible();

  const response = await listResponse;
  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No landers',
    rowLabel: (row) => String(row.name ?? ''),
  });
});
