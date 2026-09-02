import type { Campaign } from '@/api/types';

import { createDevMockCampaigns, DEV_MOCK_CUSTOMERS, DEV_MOCK_USERS } from './fixtures.ts';

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
