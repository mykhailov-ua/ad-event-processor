import { defineConfig } from '@playwright/test';

const baseURL =
  process.env.ADMIN_E2E_BASE_URL || process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:8188';

export default defineConfig({
  testDir: '.',
  testMatch: '**/*.spec.js',
  fullyParallel: true,
  workers: process.env.ADMIN_E2E_WORKERS
    ? Number(process.env.ADMIN_E2E_WORKERS)
    : process.env.CI
      ? 4
      : 2,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: [['list']],
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
      testIgnore: '**/perf/**',
    },
    {
      name: 'chromium-perf',
      testMatch: '**/perf/**/*.spec.js',
      fullyParallel: false,
      workers: 1,
      use: { browserName: 'chromium' },
    },
  ],
});
