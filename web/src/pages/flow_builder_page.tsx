import { useCallback } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { parseFlowBuilderTab, type Flow, type FlowBuilderTab } from '../helpers/flows_api.js';
import { useResource } from '../helpers/use_resource.js';
import { FlowBuilderDetail } from '../ui/flows/flow_builder_detail.js';
import { ErrorBlock } from '../ui/system/error_block.js';
import { PageSkeleton } from '../ui/system/page_skeleton.js';

export function FlowBuilderPage() {
  const { id } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const flowId = id ?? '';
  const tab = parseFlowBuilderTab(searchParams.get('tab'));
  const campaignId = searchParams.get('campaign_id')?.trim() ?? '';

  const listUrl = flowId ? `/api/v1/flows/${encodeURIComponent(flowId)}` : null;
  const { data, loading, error, reload } = useResource<Flow>(listUrl, { skip: !flowId });

  const onTabChange = useCallback(
    (next: FlowBuilderTab) => {
      const params = new URLSearchParams(searchParams);
      if (next === 'graph') {
        params.delete('tab');
      } else {
        params.set('tab', next);
      }
      setSearchParams(params, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  if (!flowId) {
    return <ErrorBlock error={new Error('missing flow id')} fallbackTitle="Invalid route" />;
  }

  if (loading && !data) {
    return <PageSkeleton rows={6} />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load flow" />;
  }

  if (!data) {
    return <ErrorBlock error={new Error('empty flow')} fallbackTitle="Flow not found" />;
  }

  return (
    <FlowBuilderDetail
      flowId={flowId}
      flow={data}
      tab={tab}
      campaignId={campaignId}
      onTabChange={onTabChange}
      onReload={reload}
    />
  );
}
