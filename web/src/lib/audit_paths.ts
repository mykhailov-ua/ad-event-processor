export type AuditHrefParams = {
  adminId?: string;
  targetId?: string;
  action?: string;
  authSource?: 'session' | 'api_key';
  apiKeyId?: string;
};

export function buildAuditHref(params: AuditHrefParams = {}): string {
  const search = new URLSearchParams();

  const adminId = params.adminId?.trim();
  if (adminId) {
    search.set('admin_id', adminId);
  }

  const targetId = params.targetId?.trim();
  if (targetId) {
    search.set('target_id', targetId);
  }

  const action = params.action?.trim();
  if (action) {
    search.set('action', action);
  }

  if (params.authSource) {
    search.set('auth_source', params.authSource);
  }

  const apiKeyId = params.apiKeyId?.trim();
  if (apiKeyId) {
    search.set('api_key_id', apiKeyId);
  }

  const query = search.toString();
  return query ? `/audit?${query}` : '/audit';
}

export function buildCampaignAuditHref(campaignId: string): string {
  return buildAuditHref({ targetId: campaignId });
}
