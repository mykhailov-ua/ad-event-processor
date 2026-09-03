export type CampaignReportDimension =
  | 'paths'
  | 'offers'
  | 'landers'
  | 'rules'
  | 'tokens'
  | 'connection'
  | 'device'
  | 'country'
  | 'default';

export function parseCampaignReportDimension(raw: string | null): CampaignReportDimension {
  switch (raw) {
    case 'paths':
    case 'offers':
    case 'landers':
    case 'rules':
    case 'tokens':
    case 'connection':
    case 'device':
    case 'country':
    case 'default':
      return raw;
    default:
      return 'country';
  }
}

export function parseCampaignReportSort(raw: string | null): string {
  const value = raw?.trim();
  if (
    value === 'name' ||
    value === 'clicks' ||
    value === 'leads' ||
    value === 'conversions' ||
    value === 'cost' ||
    value === 'revenue' ||
    value === 'profit' ||
    value === 'roi' ||
    value === 'cpc' ||
    value === 'epc' ||
    value === 'cr'
  ) {
    return value;
  }
  return 'clicks';
}

export function parseCampaignReportOrder(raw: string | null): boolean {
  return raw === 'asc' ? false : true;
}
