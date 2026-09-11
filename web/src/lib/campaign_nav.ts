import { buildExportHubHref } from '@/lib/export_hub_paths';

export function buildCampaignsDirectoryHref(params: {
  customerId?: string;
  from?: string;
  to?: string;
}): string {
  const search = new URLSearchParams();
  const customerId = params.customerId?.trim();
  if (customerId) {
    search.set('customer_id', customerId);
  }
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  const query = search.toString();
  return query ? `/campaigns?${query}` : '/campaigns';
}

export function campaignReportPath(campaignId: string): string {
  return buildExportHubHref({
    kind: 'report',
    reportKey: 'campaign-stats',
    entry: 'report-campaign-stats',
    format: 'csv',
    returnTo: campaignEditPath(campaignId),
  });
}

export function buildCampaignStatsReportHref(params: {
  campaignId: string;
  customerId?: string;
  from?: string;
  to?: string;
  granularity?: 'hour' | 'day';
}): string {
  const search = new URLSearchParams();
  search.set('campaign_id', params.campaignId.trim());
  const customerId = params.customerId?.trim();
  if (customerId) {
    search.set('customer_id', customerId);
  }
  if (params.from?.trim()) {
    search.set('from', params.from.trim());
  }
  if (params.to?.trim()) {
    search.set('to', params.to.trim());
  }
  if (params.granularity) {
    search.set('granularity', params.granularity);
  }
  return `/reports/campaign-stats?${search.toString()}`;
}

export function campaignEditPath(campaignId: string): string {
  return `/campaigns/${campaignId}/edit`;
}
