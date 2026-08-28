import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('audit export button visible with audit:read', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/audit**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'content-type': 'application/json',
        'X-Total-Count': '1',
      },
      body: JSON.stringify([
        { id: 1, action: 'login', admin_id: 'admin-1', created_at: '2026-01-01T00:00:00Z' },
      ]),
    });
  });

  await page.goto('/audit');
  await expect(page.getByRole('button', { name: 'Export CSV' })).toBeVisible();
  await expect(page.getByText('login')).toBeVisible();
});

test('audit export hidden without audit:read', async ({ page }) => {
  const noAudit = {
    ...ADMIN_USER,
    permissions: ADMIN_USER.permissions.filter((p) => p !== 'audit:read'),
  };
  await mockAuthedSession(page, noAudit);

  await page.goto('/audit');
  await expect(page.getByText('Access denied')).toBeVisible();
});
