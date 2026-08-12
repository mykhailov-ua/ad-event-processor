import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440000';
const JOB_ID = 'export-job-e2e-1';

const COMPLETED_JOB = {
  id: JOB_ID,
  customer_id: CUSTOMER_ID,
  format: 'csv',
  status: 'COMPLETED',
  bytes: 4096,
  download_url: `/api/v1/billing/exports/${JOB_ID}/download`,
  created_at: '2026-08-12T10:00:00Z',
  completed_at: '2026-08-12T10:00:05Z',
};

test('billing exports create, poll, and show download', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let createCalled = false;
  let pollCount = 0;
  let downloadCalled = false;

  await page.route('**/api/v1/billing/exports', async (route) => {
    if (route.request().method() === 'POST') {
      createCalled = true;
      await route.fulfill({
        status: 202,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ job_id: JOB_ID }),
      });
      return;
    }
    await route.continue();
  });

  await page.route(`**/api/v1/billing/exports/${JOB_ID}`, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    pollCount += 1;
    const body = pollCount < 2
      ? { ...COMPLETED_JOB, status: 'RUNNING', bytes: 0, download_url: '' }
      : COMPLETED_JOB;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(body),
    });
  });

  await page.route(`**/api/v1/billing/exports/${JOB_ID}/download`, async (route) => {
    downloadCalled = true;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'text/csv' },
      body: 'id,amount_micro,ledger_type,created_at\n',
    });
  });

  await page.goto(`/billing?customer_id=${CUSTOMER_ID}`);
  await page.getByRole('tab', { name: 'Exports' }).click();
  await expect(page.getByTestId('billing-exports-panel')).toBeVisible();

  await page.getByTestId('billing-export-create').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => createCalled).toBe(true);
  await expect(page.getByText('Export queued')).toBeVisible();
  await expect.poll(() => pollCount).toBeGreaterThanOrEqual(2);
  await expect(page.getByTestId(`billing-export-download-${JOB_ID}`)).toBeVisible();
  await expect(page.getByText('4096')).toBeVisible();

  await page.getByTestId(`billing-export-download-${JOB_ID}`).click();
  await expect.poll(() => downloadCalled).toBe(true);
});
