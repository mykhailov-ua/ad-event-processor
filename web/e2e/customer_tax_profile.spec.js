import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER, TENANT_USER } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440000';

const CUSTOMER = {
  id: CUSTOMER_ID,
  name: 'Acme Ads',
  status: 'ACTIVE',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const TAX_PROFILE = {
  customer_id: CUSTOMER_ID,
  country_code: 'US',
  tax_region: 'CA',
  tax_scheme: 'SALES_TAX',
  tax_rate_bps: 825,
};

/**
 * @param {import('@playwright/test').Page} page
 */
async function mockCustomerShell(page) {
  await page.route(`**/api/v1/customers/${CUSTOMER_ID}`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(CUSTOMER),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/wallet`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.route(`**/api/v1/campaigns*`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });
}

test('customer tax profile saves via PUT', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCustomerShell(page);

  let putBody = null;

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/tax-profile`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(TAX_PROFILE),
      });
      return;
    }
    if (route.request().method() === 'PUT') {
      putBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...TAX_PROFILE, ...putBody }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto(`/customers/${CUSTOMER_ID}`);
  await expect(page.getByTestId('customer-tax-profile')).toBeVisible();
  await page.locator('#tax-country').fill('DE');
  await page.locator('#tax-region').fill('BY');
  await page.locator('#tax-scheme').fill('VAT');
  await page.locator('#tax-rate-bps').fill('1900');
  await page.getByRole('button', { name: 'Save tax profile' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => putBody).not.toBeNull();
  expect(putBody.country_code).toBe('DE');
  expect(putBody.tax_region).toBe('BY');
  expect(putBody.tax_scheme).toBe('VAT');
  expect(putBody.tax_rate_bps).toBe(1900);
});

test('tenant user sees read-only tax profile without save form', async ({ page }) => {
  const tenantCustomerId = 'cust-own';
  const tenantCustomer = { ...CUSTOMER, id: tenantCustomerId };

  await mockAuthedSession(page, TENANT_USER);

  await page.route(`**/api/v1/customers/${tenantCustomerId}`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(tenantCustomer),
    });
  });

  await page.route(`**/api/v1/customers/${tenantCustomerId}/wallet`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.route(`**/api/v1/customers/${tenantCustomerId}/tax-profile`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ ...TAX_PROFILE, customer_id: tenantCustomerId }),
    });
  });

  await page.goto(`/customers/${tenantCustomerId}`);
  const taxSection = page.getByTestId('customer-tax-profile');
  await expect(taxSection).toBeVisible();
  await expect(taxSection.getByText('US', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save tax profile' })).toHaveCount(0);
  await expect(page.locator('#tax-country')).toHaveCount(0);
});
