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

export function campaignReportPath(_campaignId: string): string | null {
  return null;
}

export function campaignEditPath(campaignId: string): string {
  return `/campaigns/${campaignId}/edit`;
}
