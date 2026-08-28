import { useCallback } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { parseCampaignDetailTab, type Campaign, type CampaignDetailTab } from '../helpers/campaigns_api.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { can } from '../helpers/permissions.js';
import { useResource } from '../helpers/use_resource.js';
import { CampaignDetail } from '../ui/campaigns/campaign_detail.js';
import { ErrorBlock } from '../ui/system/error_block.js';
import { PageSkeleton } from '../ui/system/page_skeleton.js';

export function CampaignDetailPage() {
  const { id } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const campaignId = id ?? '';
  const masked = can(permissions, 'campaigns:read:masked') && !can(permissions, 'campaigns:read');

  const tab = parseCampaignDetailTab(searchParams.get('tab'));
  const effectiveTab =
    masked && !['overview', 'stats', 'config'].includes(tab) ? 'overview' : tab;

  const listUrl = campaignId ? `/api/v1/campaigns/${encodeURIComponent(campaignId)}` : null;
  const { data, loading, error, reload } = useResource<Campaign>(listUrl, {
    skip: !campaignId,
  });

  const onTabChange = useCallback(
    (next: CampaignDetailTab) => {
      const params = new URLSearchParams(searchParams);
      if (next === 'overview') {
        params.delete('tab');
      } else {
        params.set('tab', next);
      }
      setSearchParams(params, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  if (!campaignId) {
    return <ErrorBlock error={new Error('missing campaign id')} fallbackTitle="Invalid route" />;
  }

  if (loading && !data) {
    return <PageSkeleton rows={6} />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load campaign" />;
  }

  if (!data) {
    return <ErrorBlock error={new Error('empty campaign')} fallbackTitle="Campaign not found" />;
  }

  if (data.customer_id) {
    touchCustomerContext(data.customer_id);
  }

  return (
    <CampaignDetail
      campaignId={campaignId}
      campaign={data}
      tab={effectiveTab}
      masked={masked}
      onTabChange={onTabChange}
      onReload={reload}
    />
  );
}
