/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession } from './helpers.js';

const AUDIT_USER = {
  id: 'audit-1',
  email: 'audit@test.local',
  role: 'A',
  customer_id: '',
  permissions: ['audit:read', 'settings:read'],
};

const AUDIT_ROWS = [
  {
    id: 1,
    action: 'CREATE_CAMPAIGN',
    target_type: 'campaign',
    target_id: '550e8400-e29b-41d4-a716-446655440000',
    admin_id: 'admin-1',
    created_at: '2026-08-12T10:00:00Z',
  },
];

test('audit export downloads CSV and shows truncated banner', async ({ page }) => {
  await mockAuthedSession(page, AUDIT_USER);

  let exportUrl = '';
  await page.route('**/api/v1/audit**', async (route) => {
    const url = route.request().url();
    if (url.includes('/export')) {
      exportUrl = url;
      await route.fulfill({
        status: 200,
        headers: {
          'content-type': 'text/csv',
          'X-Export-Truncated': 'true',
          'X-Next-Cursor': '500',
          'X-Export-Bytes': '12',
        },
        body: 'id,admin_id,action\n1,admin,TEST\n',
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: {
        'content-type': 'application/json',
        'X-Total-Count': '1',
      },
      body: JSON.stringify(AUDIT_ROWS),
    });
  });

  await page.goto('/audit');
  await expect(page.getByTestId('audit-export-csv')).toBeVisible();
  await page.getByTestId('audit-export-csv').click();
  await expect(page.getByTestId('audit-export-truncated')).toBeVisible();
  expect(exportUrl).toContain('redact_pii=true');
  expect(exportUrl).toContain('format=csv');
});

test('audit export hidden without audit:read', async ({ page }) => {
  await mockAuthedSession(page, {
    ...AUDIT_USER,
    permissions: ['settings:read'],
  });

  await page.route('**/api/v1/audit**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json', 'X-Total-Count': '0' },
      body: '[]',
    });
  });

  await page.goto('/audit');
  await expect(page.getByTestId('audit-export-csv')).toHaveCount(0);
});
