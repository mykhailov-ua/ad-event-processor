import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER, PUBLISHER_USER, BUYER_USER } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440001';
const EXPORT_JOB_ID = 'cpa-m2-export-held-out';

const LEDGER_CSV = [
  'id,amount_micro,ledger_type,created_at',
  '99,2500000,TOPUP,2026-08-01T12:00:00Z',
  '',
].join('\n');

test.describe('CPA held-out - ledger export', () => {
  test('ledger CSV download has header + data row with TOPUP', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    let downloadBody = '';

    await page.route('**/api/v1/billing/exports', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 202,
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({ job_id: EXPORT_JOB_ID }),
        });
        return;
      }
      await route.continue();
    });

    await page.route(`**/api/v1/billing/exports/${EXPORT_JOB_ID}`, async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          id: EXPORT_JOB_ID,
          customer_id: CUSTOMER_ID,
          format: 'csv',
          status: 'COMPLETED',
          bytes: LEDGER_CSV.length,
          download_url: `/api/v1/billing/exports/${EXPORT_JOB_ID}/download`,
          created_at: '2026-08-01T10:00:00Z',
          completed_at: '2026-08-01T10:00:05Z',
        }),
      });
    });

    await page.route(`**/api/v1/billing/exports/${EXPORT_JOB_ID}/download`, async (route) => {
      downloadBody = LEDGER_CSV;
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'text/csv' },
        body: LEDGER_CSV,
      });
    });

    await page.goto(`/billing?customer_id=${CUSTOMER_ID}&tab=exports`);
    await expect(page.getByTestId('billing-exports-panel')).toBeVisible();
    await page.getByTestId('billing-export-create').click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
    await expect(page.getByTestId(`billing-export-download-${EXPORT_JOB_ID}`)).toBeVisible({
      timeout: 10000,
    });
    await page.getByTestId(`billing-export-download-${EXPORT_JOB_ID}`).click();

    await expect.poll(() => downloadBody.length).toBeGreaterThan(0);
    const lines = downloadBody.trim().split('\n');
    expect(lines.length).toBeGreaterThanOrEqual(2);
    expect(lines[0]).toContain('ledger_type');
    expect(downloadBody).toContain('TOPUP');
    expect(downloadBody).toContain('2500000');
  });
});

test.describe('CPA held-out - campaign PATCH honesty', () => {
  test.fixme('budget save persists after reload', async ({ page }) => {
    await page.goto('/campaigns/00000000-0000-4000-8000-000000000001');
    await page.getByTestId('campaign-budget-total').fill('1000');
    await page.getByRole('button', { name: 'Save' }).click();
    await page.reload();
    await expect(page.getByTestId('campaign-budget-total')).toHaveValue('1000');
  });
});

test.describe('CPA held-out - report actions', () => {
  test('pause campaign POST before success toast', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);
    const CAMPAIGN_ID = '550e8400-e29b-41d4-a716-446655440099';
    let sawPause = false;
    await page.route('**/api/v1/reports/source-quality**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          rows: [
            {
              placement_id: 'sub-a',
              campaign_id: CAMPAIGN_ID,
              clicks: 10,
              ivt_rate: 0.2,
              roi_pct: 1,
            },
          ],
          freshness: { as_of: '2026-08-01T00:00:00Z', consistency: 'eventual', stale: false },
        }),
      });
    });
    await page.route(`**/api/v1/selfserve/campaigns/${CAMPAIGN_ID}/pause`, async (route) => {
      sawPause = true;
      await route.fulfill({ status: 200, body: '{}' });
    });
    await page.goto('/reports/source-quality?customer_id=550e8400-e29b-41d4-a716-446655440001');
    await page.getByTestId('report-row-actions-toggle').click();
    await page.getByTestId('report-action-pause').click();
    await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
    expect(sawPause).toBe(true);
  });
});

test.describe('CPA held-out - integration kit', () => {
  test('click URL includes dmr=1 when toggled', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);
    const CAMPAIGN_ID = '550e8400-e29b-41d4-a716-446655440000';

    await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}`, async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          id: CAMPAIGN_ID,
          name: 'Kit',
          status: 'ACTIVE',
          customer_id: 'cust-1',
        }),
      });
    });
    await page.route('**/api/v1/ops/doctor', async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({
          tracking_domain: 'trk.example.com',
          click_url_template: 'https://trk.example.com/click?campaign_id={campaign_id}',
        }),
      });
    });
    await page.route('**/api/v1/settings/platform', async (route) => {
      await route.fulfill({ status: 200, body: JSON.stringify({ config: {} }) });
    });
    await page.route('**/api/v1/postbacks/**', async (route) => {
      await route.fulfill({ status: 200, body: '[]' });
    });
    await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}/dashboard**`, async (route) => {
      await route.fulfill({ status: 200, body: JSON.stringify({ kpis: {} }) });
    });
    await page.route('**/api/v1/buyer/portfolio**', async (route) => {
      await route.fulfill({ status: 200, body: JSON.stringify({ items: [] }) });
    });

    await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=tracking`);
    await page.getByTestId('integration-dmr-toggle').check();
    await expect(page.getByTestId('integration-click-url')).toContainText('dmr=1');
  });
});

test.describe('CPA held-out - publisher scope', () => {
  test('publisher dashboard loads with scoped fixture', async ({ page }) => {
    await mockAuthedSession(page, PUBLISHER_USER);

    await page.route('**/api/v1/publisher/dashboard**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          seller_id: 'seller-e2e',
          publisher_account_id: 'acct-e2e',
          from: '2026-08-01T00:00:00Z',
          to: '2026-08-08T00:00:00Z',
          kpis: { impressions: 1200, fill_rate: 0.12, ecpm_micro: 450000, ivt_rate: 0.01 },
          placements: [
            {
              placement_id: 'seller-e2e/banner-1',
              impressions: 1200,
              clicks: 144,
              fill_rate: 0.12,
              revenue_micro: 540000,
              ecpm_micro: 450000,
            },
          ],
        }),
      });
    });

    await page.route('**/api/v1/publisher/statements**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0 }),
      });
    });

    await page.route('**/api/v1/supply/validation**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          sellers_json_valid: true,
          sellers_checksum_sha256: 'abc123',
          sellers_count: 1,
          ads_txt_valid: true,
          ads_txt_checksum_sha256: 'def456',
          ads_txt_line_count: 2,
          issues: [],
        }),
      });
    });

    await page.goto('/publisher');
    await expect(page.getByTestId('publisher-portal')).toBeVisible();
    await expect(page.getByText('seller-e2e')).toBeVisible();
    await expect(page.getByText('1200')).toBeVisible();
  });

  test('publisher nav hides campaigns', async ({ page }) => {
    await mockAuthedSession(page, PUBLISHER_USER);
    await page.route('**/api/v1/publisher/dashboard**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          seller_id: 'seller-e2e',
          from: '2026-08-01T00:00:00Z',
          to: '2026-08-08T00:00:00Z',
          kpis: { impressions: 0, fill_rate: 0, ecpm_micro: 0, ivt_rate: 0 },
          placements: [],
        }),
      });
    });
    await page.route('**/api/v1/publisher/statements**', async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ items: [], total: 0 }),
      });
    });
    await page.route('**/api/v1/supply/validation**', async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({
          sellers_json_valid: true,
          sellers_checksum_sha256: 'x',
          sellers_count: 0,
          ads_txt_valid: true,
          ads_txt_checksum_sha256: 'y',
          ads_txt_line_count: 0,
        }),
      });
    });

    await page.goto('/publisher');
    await expect(page.getByRole('link', { name: 'Campaigns' })).toHaveCount(0);
  });
});

test.describe('CPA held-out - self-serve portal', () => {
  test('billing statement and payment intent in selfserve shell', async ({ page }) => {
    await mockAuthedSession(page, BUYER_USER);

    await page.route('**/api/v1/dashboards/buyer**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ campaigns: [], totals: {} }),
      });
    });

    await page.route('**/api/v1/selfserve/invoices**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ invoices: [], total: 0 }),
      });
    });

    await page.route('**/api/v1/selfserve/billing/statement*', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          opening_balance_micro: 1_000_000,
          closing_balance_micro: 2_500_000,
          currency: 'USD',
          period: { from: '2026-08-01', to: '2026-08-31' },
        }),
      });
    });

    await page.route('**/api/v1/selfserve/payment-intents', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          intent_id: 'pi-selfserve-e2e',
          status: 'PAYMENT_INTENT_STATUS_PENDING_PROVIDER',
          checkout_url: 'https://checkout.example/pi-selfserve-e2e',
        }),
      });
    });

    await page.goto('/selfserve/billing');
    await expect(page.getByTestId('selfserve-shell')).toBeVisible();
    await expect(page.getByTestId('billing-selfserve-panel')).toBeVisible();
    await expect(page.getByRole('link', { name: 'Customers' })).toHaveCount(0);
    await expect(page.getByRole('link', { name: 'Ops' })).toHaveCount(0);

    await page.getByTestId('billing-statement-load').click();
    await expect(page.getByText('1.00 USD')).toBeVisible();

    await page.getByTestId('billing-topup-amount').fill('25.00');
    await page.getByTestId('billing-topup-submit').click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByRole('button', { name: 'Confirm' }).click();
    await expect(page.getByTestId('billing-topup-checkout-link')).toBeVisible();
  });
});

test.describe('CPA held-out - ops consolidation', () => {
  test('consent browser is read-only list', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route('**/api/v1/ops/consent/proofs**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          items: [
            {
              id: 99,
              user_id_hash: 'deadbeef',
              purposes: 1,
              source: 'cmp',
              recorded_at: '2026-08-12T10:00:00Z',
              ad_storage: true,
              analytics_storage: false,
            },
          ],
        }),
      });
    });

    await page.goto('/ops/consent');
    await expect(page.getByTestId('consent-proof-row-99')).toContainText('cmp');
    await expect(page.getByRole('button', { name: 'Delete' })).toHaveCount(0);
    await expect(page.getByTestId('consent-submit')).toHaveCount(0);
  });

  test('unified DLQ inbox retries with source body', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    let retryBody = null;

    await page.route('**/api/v1/ops/dlq/inbox**', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          items: [
            {
              id: '11',
              source: 'capi',
              campaign_id: 'camp-m8',
              event_type: 'conversion',
              error: '401',
              failed_at: '2026-08-12T10:00:00Z',
              status: 'FAILED',
            },
          ],
        }),
      });
    });

    await page.route('**/api/v1/ops/dlq/inbox/11/retry', async (route) => {
      retryBody = route.request().postDataJSON();
      await route.fulfill({ status: 202, body: '' });
    });

    await page.goto('/ops/dlq');
    await page.getByTestId('dlq-inbox-retry-capi-11').click();
    await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
    await expect.poll(() => retryBody?.source).toBe('capi');
  });

  test('stale dashboard shows affected campaigns', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route('**/api/v1/ops/incidents', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          partial: false,
          shards: [],
          outbox: { pending: 0 },
          stale_dashboard: true,
          affected_campaigns: [
            {
              campaign_id: '550e8400-e29b-41d4-a716-446655440088',
              name: 'Stale KPI Camp',
            },
          ],
        }),
      });
    });

    await page.route('**/api/v1/ops/doctor', async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ overall: 'ok', checks: [] }),
      });
    });
    await page.route('**/api/v1/ops/dashboard/summary', async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ outbox_pending: 0, emergency_breaker: 'closed', services: [] }),
      });
    });
    await page.route('**/api/v1/ops/rum', async (route) => {
      await route.fulfill({ status: 200, body: JSON.stringify({ events: [] }) });
    });
    await page.route('**/api/v1/dashboards/operator', async (route) => {
      await route.fulfill({ status: 200, body: JSON.stringify(null) });
    });

    await page.goto('/ops');
    await expect(page.getByTestId('ops-stale-dashboard')).toBeVisible();
    await expect(page.getByText('Stale KPI Camp')).toBeVisible();
  });
});
