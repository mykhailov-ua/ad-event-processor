/** harness=mock_api — integration schemas page; does not prove Postgres integration_schemas. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const SCHEMAS = [
  {
    id: 'schema-1',
    name: 'traffic_propellerads',
    version: 1,
    kind: 'traffic_source',
    schema: {},
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
];

const CATALOG = [
  {
    name: 'traffic_propellerads',
    file: 'traffic_propellerads.yaml',
    version: 1,
    category: 'traffic',
    kind: 'traffic_source',
  },
];

test('integrations schemas page lists stored schemas', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/integration/schemas', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(SCHEMAS),
    });
  });

  await page.route('**/api/v1/integration/templates', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(CATALOG),
    });
  });

  await page.goto('/integrations/schemas');
  await expect(page.getByRole('table', { name: 'Integration schemas' })).toContainText('traffic_propellerads');
  await expect(page.getByTestId('integration-template-import')).toContainText('traffic_propellerads');
});

test('author custom schema via POST /integration/schemas', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let createPosted = false;

  await page.route('**/api/v1/integration/schemas', async (route) => {
    if (route.request().method() === 'POST') {
      createPosted = true;
      const body = route.request().postDataJSON();
      expect(body.name).toBe('custom_smoke_outbound');
      expect(body.version).toBe(1);
      expect(body.schema.url_template).toContain('click_id');
      await route.fulfill({
        status: 201,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          id: 'schema-new-1',
          name: body.name,
          version: body.version,
          kind: 'outbound_postback',
          schema: body.schema,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        }),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(SCHEMAS),
    });
  });

  await page.route('**/api/v1/integration/templates', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(CATALOG),
    });
  });

  await page.goto('/integrations/schemas');
  await page.getByTestId('schema-author-name').fill('custom_smoke_outbound');
  await page.getByTestId('schema-author-submit').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => createPosted).toBe(true);
  await expect(page.getByTestId('schema-row-schema-new-1')).toContainText('custom_smoke_outbound');
});

test('billing page shows fleet summary for shards:read admin', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/summary', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        invoiced_mtd_micro: 250_000_000,
        invoice_count_mtd: 3,
        undelivered_invoice_notifications: 1,
        customers_with_spend_in_month: 2,
      }),
    });
  });

  await page.route('**/api/v1/customers/*/wallet', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.goto('/billing');
  await expect(page.getByTestId('billing-summary-panel')).toBeVisible();
  await expect(page.getByTestId('billing-summary-panel')).toContainText('250.00');
  await expect(page.getByTestId('billing-summary-panel')).toContainText('3');
});
