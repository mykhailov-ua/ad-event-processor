export const ADMIN_USER = {
  id: 'admin-1',
  email: 'admin@test.local',
  role: 'A',
  customer_id: '',
  permissions: [
    'customers:read',
    'customers:write',
    'campaigns:read',
    'campaigns:write',
    'campaigns:pause',
    'shards:read',
    'shards:write',
    'settings:read',
    'settings:write',
    'billing:read',
    'billing:write',
    'audit:read',
    'rtb:read',
    'rtb:write',
    'blacklist:read',
    'users:write',
  ],
};

export const BUYER_USER = {
  id: 'buyer-1',
  email: 'buyer@test.local',
  role: 'B',
  customer_id: 'cust-1',
  permissions: ['campaigns:read:masked', 'customers:read'],
};

export const TENANT_USER = {
  id: 'user-1',
  email: 'user@test.local',
  role: 'U',
  customer_id: 'cust-own',
  permissions: ['customers:read', 'campaigns:read'],
};

export const TEAM_LEAD_USER = {
  id: 'tl-1',
  email: 'lead@test.local',
  role: 'TL',
  customer_id: 'cust-team-1',
  permissions: [
    'campaigns:read',
    'campaigns:write',
    'billing:read',
    'customers:read',
    'users:write',
  ],
};

export const PUBLISHER_USER = {
  id: 'pub-1',
  email: 'publisher@test.local',
  role: 'P',
  customer_id: 'cust-pub-1',
  permissions: ['supply:read:scoped', 'customers:read'],
};

export const PLATFORM_VIEW = {
  config: {
    tracking_domain: 'track.example',
    default_currency: 'USD',
    timezone: 'UTC',
    ingress_schema: 'ad_event_processor_native',
    telemetry_enabled: true,
    profile: 'single_vps',
    edge_xdp: false,
    network_interface: 'eth0',
    stripe: { enabled: false },
  },
  bootstrap_complete: true,
  restart_required: [],
};

export function installDialogAutoAccept(page) {
  page.on('dialog', async (dialog) => {
    await dialog.accept();
  });
}

export async function mockAuthedSession(page, user = ADMIN_USER) {
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

  await page.route('**/api/v1/eula', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ required: false }),
    });
  });

  await mockDirectoryApis(page);
}

export async function mockDirectoryApis(page) {
  await page.route('**/api/v1/customers**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/customers\/[^/?]/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.route('**/api/v1/campaigns**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/campaigns\/[^/?]+/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });
}

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

export async function loginViaForm(page, email = 'admin@test.local', password = 'secret') {
  await page.goto('/login');
  await page.fill('#login-email', email);
  await page.fill('#login-password', password);
  await Promise.all([
    page.waitForURL(/\/customers/, { timeout: 15_000 }),
    page.getByRole('button', { name: 'Sign in' }).click(),
  ]);
}

export async function mockEmptyCampaigns(page) {
  await page.route('**/api/v1/campaigns**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/campaigns\/[^/?]+/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });
}

export async function mockEmptyCustomers(page) {
  await page.route('**/api/v1/customers**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/customers\/[^/?]/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });
}

export async function mockPlatformSettings(page, view = PLATFORM_VIEW) {
  await page.route('**/api/v1/settings/platform**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(view),
      });
      return;
    }
    if (method === 'PATCH') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(view),
      });
      return;
    }
    await route.continue();
  });
}

export async function mockOpsDashboard(page, summary = {}) {
  const body = {
    outbox_pending: 3,
    rps_estimate: 1200,
    drift_alert: false,
    emergency_breaker: 'closed',
    services: [
      { name: 'Management', status: 'ok' },
      { name: 'ClickHouse', status: 'ok' },
    ],
    ...summary,
  };

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(body),
    });
  });

  await page.route('**/api/v1/ops/doctor**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        overall: 'ok',
        checks: [{ id: 'mgmt', status: 'ok', message: 'Management' }],
      }),
    });
  });
}

export async function mockReportCatalog(page, rows = []) {
  await page.route('**/api/v1/reports/catalog**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ rows }),
    });
  });
}

export const CATALOG_SPEND_VELOCITY = {
  key: 'spend-velocity',
  title: 'Spend velocity',
  description: 'Spend rate over time',
  category: 'commercial',
};
