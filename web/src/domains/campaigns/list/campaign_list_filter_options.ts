import type { CampaignsListFilterOption } from './campaigns_list_filter_select';

const ALL_COUNTRIES_VALUE = '__all__';
const ALL_OWNERS_VALUE = '__all__';

export type CampaignListFacetOwnerInput = {
  user_id: string;
  email?: string | null;
};

export function buildCampaignListCountryOptions(
  countries: readonly string[],
  appliedCountry?: string,
): CampaignsListFilterOption[] {
  const codes = new Set<string>();
  for (const code of countries) {
    const trimmed = code?.trim();
    if (trimmed) {
      codes.add(trimmed);
    }
  }
  const applied = appliedCountry?.trim();
  if (applied) {
    codes.add(applied);
  }

  return [
    { value: ALL_COUNTRIES_VALUE, label: 'All countries' },
    ...[...codes].sort().map((code) => ({ value: code, label: code })),
  ];
}

export function buildCampaignListOwnerOptions(
  owners: readonly CampaignListFacetOwnerInput[],
  appliedOwnerUserId?: string,
): CampaignsListFilterOption[] {
  const options: CampaignsListFilterOption[] = [{ value: ALL_OWNERS_VALUE, label: 'All owners' }];
  const seen = new Set<string>([ALL_OWNERS_VALUE]);

  const addOwner = (userId: string, label: string) => {
    if (!userId || seen.has(userId)) {
      return;
    }
    seen.add(userId);
    options.push({ value: userId, label: label.trim() || userId.slice(0, 8) });
  };

  for (const owner of owners) {
    const userId = owner.user_id ?? '';
    addOwner(userId, owner.email?.trim() || userId);
  }
  if (appliedOwnerUserId) {
    const appliedOwner = owners.find((owner) => owner.user_id === appliedOwnerUserId);
    addOwner(appliedOwnerUserId, appliedOwner?.email?.trim() || appliedOwnerUserId);
  }
  return options;
}

export function buildCampaignListOwnerEmailById(
  owners: readonly CampaignListFacetOwnerInput[],
): Record<string, string> {
  const map: Record<string, string> = {};
  for (const owner of owners) {
    const userId = owner.user_id;
    const email = owner.email?.trim();
    if (userId && email) {
      map[userId] = email;
    }
  }
  return map;
}
