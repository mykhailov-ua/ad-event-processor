import type { Campaign } from '@/api/types';

import { createDevMockCampaigns, DEV_MOCK_CUSTOMERS, DEV_MOCK_USERS } from './fixtures.ts';

// T0 in-memory mock state (admin_dev=1). Singleton survives SPA navigations until resetDevMockStore().
// Not used when api/client.ts reaches the real control plane.
export type DevMockStore = {
  campaigns: Campaign[];
};

let store: DevMockStore | undefined;

export function devMockStore(): DevMockStore {
  if (!store) {
    store = { campaigns: createDevMockCampaigns() };
  }
  return store;
}

export function resetDevMockStore(): void {
  store = { campaigns: createDevMockCampaigns() };
}

export { DEV_MOCK_CUSTOMERS, DEV_MOCK_USERS };
