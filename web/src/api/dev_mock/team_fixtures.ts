import { DEV_MOCK_CUSTOMERS, DEV_MOCK_USERS } from './fixtures.ts';
import { devMockStore } from './store.ts';
import { devMockIso, parseLimitOffset, slicePage, usdToMicro } from './fixture_helpers.ts';
import { seedDeterministicUuid } from './seed_uuid.ts';

export function devMockTeamOverview(customerId: string | undefined) {
  const customer = DEV_MOCK_CUSTOMERS.find((row) => row.id === customerId) ?? DEV_MOCK_CUSTOMERS[0];
  const campaigns = devMockStore().campaigns.filter((row) => row.customer_id === customer.id);
  const members = DEV_MOCK_USERS.map((user, index) => ({
    user_id: user.id,
    email: user.email,
    role: index === 0 ? 'admin' : index === 1 ? 'buyer' : 'analyst',
    campaigns_owned: campaigns.filter((row) => row.owner_user_id === user.id).length,
    created_at: devMockIso(40 + index),
    created_at_display: devMockIso(40 + index),
    is_blocked: false,
    spend_cap_micro: usdToMicro(50_000 + index * 10_000),
  }));
  return {
    customer_id: customer.id,
    customer_name: customer.name,
    cost_center: `CC-${customer.name.slice(0, 3).toUpperCase()}`,
    balance_micro: usdToMicro(18_420.55),
    currency: 'USD',
    license: {
      state: 'ACTIVE',
      valid_until: devMockIso(-120),
      plan_code: 'enterprise',
    },
    members,
  };
}

export function devMockTeamMembersList() {
  const campaigns = devMockStore().campaigns;
  const items = DEV_MOCK_USERS.map((user, index) => ({
    user_id: user.id,
    email: user.email,
    role: index === 0 ? 'admin' : index === 1 ? 'buyer' : 'analyst',
    campaigns_owned: campaigns.filter((row) => row.owner_user_id === user.id).length,
    created_at: devMockIso(30 + index),
    created_at_display: devMockIso(30 + index),
    is_blocked: index === 2 && false,
    spend_cap_micro: usdToMicro(40_000 + index * 8_500),
  }));
  return { items, total: items.length };
}

export function devMockTeamBudgetApprovals(url: URL) {
  const { limit, offset } = parseLimitOffset(url, 100);
  const campaigns = devMockStore().campaigns.slice(0, 8);
  const items = campaigns.map((campaign, index) => ({
    id: seedDeterministicUuid('budget_approval', index + 1),
    user_id: campaign.owner_user_id ?? DEV_MOCK_USERS[1].id,
    campaign_id: campaign.id,
    requested_budget_micro: usdToMicro(8_000 + index * 1_250),
    previous_budget_micro: usdToMicro(5_000 + index * 900),
    status: index % 4 === 0 ? 'pending' : index % 4 === 1 ? 'approved' : 'denied',
    created_at: devMockIso(index % 14),
    created_at_display: devMockIso(index % 14),
  }));
  const page = slicePage(items, limit, offset);
  return { ...page, limit, offset };
}
