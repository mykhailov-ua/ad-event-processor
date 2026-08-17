/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440000';
const JOB_ID = 'export-job-e2e-1';

const COMPLETED_JOB = {
  id: JOB_ID,
  customer_id: CUSTOMER_ID,
  format: 'csv',
  status: 'COMPLETED',
  bytes: 0,
  download_url: `/api/v1/billing/exports/${JOB_ID}/download`,
  created_at: '2026-08-12T10:00:00Z',
  completed_at: '2026-08-12T10:00:05Z',
};

const LEDGER_CSV = [
  'id,amount_micro,ledger_type,created_at',
  '42,1500000,TOPUP,2026-08-12T10:00:00Z',
  '',
].join('\n');

COMPLETED_JOB.bytes = LEDGER_CSV.length;

test('billing exports create, poll, and show download', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let createCalled = false;
  let pollCount = 0;
  let downloadCalled = false;
  let downloadBody = '';

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
    downloadBody = LEDGER_CSV;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'text/csv' },
      body: LEDGER_CSV,
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
  await expect(page.getByText(String(COMPLETED_JOB.bytes))).toBeVisible();

  await page.getByTestId(`billing-export-download-${JOB_ID}`).click();
  await expect.poll(() => downloadCalled).toBe(true);
  const csvLines = downloadBody.trim().split('\n');
  expect(csvLines.length).toBeGreaterThanOrEqual(2);
  expect(csvLines[0]).toBe('id,amount_micro,ledger_type,created_at');
  expect(downloadBody).toContain('TOPUP');
  expect(downloadBody).toContain('1500000');
});
