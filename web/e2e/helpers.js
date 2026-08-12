/** Shared Playwright API mocks for admin e2e. */

export const ADMIN_USER = {
  id: 'admin-1',
  email: 'admin@test.local',
  role: 'A',
  customer_id: '',
  permissions: [
    'customers:read',
    'campaigns:read',
    'shards:read',
    'shards:write',
    'settings:read',
    'settings:write',
    'billing:read',
    'billing:write',
    'customers:write',
    'campaigns:write',
    'campaigns:pause',
  ],
};

export const BUYER_USER = {
  id: 'buyer-1',
  email: 'buyer@test.local',
  role: 'B',
  customer_id: 'cust-1',
  permissions: ['campaigns:read:masked'],
};

export const TENANT_USER = {
  id: 'user-1',
  email: 'user@test.local',
  role: 'U',
  customer_id: 'cust-own',
  permissions: ['customers:read', 'campaigns:read'],
};

/**
 * @param {import('@playwright/test').Page} page
 * @param {object} user
 */
export async function mockAuthedSession(page, user) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'content-type': 'application/json',
        'X-CSRF-Token': 'e2e-csrf-token',
      },
      body: JSON.stringify({
        id: user.id,
        email: user.email,
        role: user.role,
        customer_id: user.customer_id,
        permissions: user.permissions,
      }),
    });
  });

  await page.route('**/api/v1/meta', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ version: 'e2e', bootstrap_complete: true }),
    });
  });
}

/**
 * @param {import('@playwright/test').Page} page
 */
export async function mockLoginSuccess(page, user = ADMIN_USER) {
  await page.route('**/api/v1/auth/login', async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'content-type': 'application/json',
        'X-CSRF-Token': 'e2e-csrf-token',
      },
      body: JSON.stringify({ user }),
    });
  });
  await mockAuthedSession(page, user);
}
