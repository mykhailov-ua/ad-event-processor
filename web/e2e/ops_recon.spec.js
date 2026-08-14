/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession } from './helpers.js';

const RECON_USER = {
  id: 'recon-1',
  email: 'recon@test.local',
  role: 'A',
  customer_id: '',
  permissions: ['audit:read', 'shards:read'],
};

const RECON_ROWS = [
  {
    service: 'management',
    id: 10,
    period_start: '2026-08-01T00:00:00Z',
    period_end: '2026-08-02T00:00:00Z',
    status: 'COMPLETED',
    discrepancies_found: 2,
    created_at: '2026-08-02T01:00:00Z',
  },
  {
    service: 'payment',
    id: 11,
    period_start: '2026-08-01T00:00:00Z',
    period_end: '2026-08-02T00:00:00Z',
    status: 'FAILED',
    findings_count: 1,
    created_at: '2026-08-02T02:00:00Z',
  },
];

test('ops recon lists runs with pagination header', async ({ page }) => {
  await mockAuthedSession(page, RECON_USER);

  await page.route('**/api/v1/recon/runs**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'content-type': 'application/json',
        'X-Total-Count': String(RECON_ROWS.length),
      },
      body: JSON.stringify(RECON_ROWS),
    });
  });

  await page.goto('/ops/recon');
  await expect(page.getByTestId('ops-recon-view')).toBeVisible();
  await expect(page.getByTestId('ops-recon-table')).toBeVisible();
  await expect(page.getByTestId('ops-recon-table').getByRole('cell', { name: 'management' })).toBeVisible();
  await expect(page.getByTestId('ops-recon-table').getByRole('cell', { name: 'payment' })).toBeVisible();
  await expect(page.getByText('COMPLETED', { exact: true })).toBeVisible();
});
