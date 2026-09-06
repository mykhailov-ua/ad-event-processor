import { DEV_MOCK_USERS } from './fixtures.ts';
import { devMockIso, parseLimitOffset, slicePage } from './fixture_helpers.ts';
import { seedAuditTargetId } from './fixture_names.ts';

const AUDIT_ACTIONS = [
  'campaign.update',
  'campaign.pause',
  'customer.create',
  'settings.patch',
  'billing.invoice.finalize',
  'fraud.preset.update',
  'rtb.deal.create',
  'team.member.invite',
] as const;

const AUDIT_TARGETS = [
  'campaign',
  'customer',
  'platform_settings',
  'invoice',
  'fraud_preset',
  'rtb_deal',
  'team_member',
] as const;

export function devMockAuditList(url: URL) {
  const { limit, offset } = parseLimitOffset(url);
  const items = Array.from({ length: 40 }, (_, index) => {
    const admin = DEV_MOCK_USERS[index % DEV_MOCK_USERS.length];
    const action = AUDIT_ACTIONS[index % AUDIT_ACTIONS.length];
    const targetType = AUDIT_TARGETS[index % AUDIT_TARGETS.length];
    return {
      id: 10_000 + index,
      admin_id: admin.id,
      action,
      target_type: targetType,
      target_id: seedAuditTargetId(targetType, index + 1),
      changes: { field: 'status', from: 'ACTIVE', to: 'PAUSED' },
      metadata: { source: 'dev_mock' },
      is_masked: index % 9 === 0,
      created_at: devMockIso(index % 25, index),
      created_at_display: devMockIso(index % 25, index),
    };
  });
  const page = slicePage(items, limit, offset);
  return { ...page, limit, offset };
}
