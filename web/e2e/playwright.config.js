import { defineConfig } from '@playwright/test';

const stackE2E = process.env.ADMIN_STACK_E2E === '1';
const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:4173';

export default defineConfig({
  testDir: '.',
  timeout: 45_000,
  retries: process.env.CI ? 1 : 0,
  use: {
    baseURL,
    headless: true,
  },
  webServer: stackE2E
    ? undefined
    : {
        command: 'node ../scripts/preview.mjs',
        port: 4173,
        reuseExistingServer: !process.env.CI,
      },
});
