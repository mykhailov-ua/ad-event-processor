// L3 editor tools tabs: integration/fraud GET gated by tab; context uses lazy loadKey; fraud PATCH refresh coalesced while fetching.
import { useState } from 'react';

import { getCampaignFraud, getCampaignIntegrationPanel } from '@/api/campaigns_api';
import type { CampaignFraudConfig } from '@/api/types';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

import type { CampaignEditorToolsTab } from '@/domains/campaigns/editor/campaign_editor_tools';
import { useCampaignEditorContextLoad } from '@/domains/campaigns/editor/use_campaign_editor_context_load';
import { useCampaignFraudPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_fraud_panel_workspace';
import { useCampaignIntegrationPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_integration_panel_workspace';
import { useCampaignOpsPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_ops_panel_workspace';

export function useCampaignEditorToolsLoad(campaignId: string) {
  const [tab, setTab] = useState<CampaignEditorToolsTab>('integration');
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const integrationResource = useResource(
    (signal) => {
      if (tab !== 'integration') {
        return Promise.resolve(undefined);
      }
      return getCampaignIntegrationPanel(campaignId, signal);
    },
    [campaignId, tab]
  );

  const fraudResource = useResource(
    (signal) => {
      if (tab !== 'fraud') {
        return Promise.resolve(undefined);
      }
      return getCampaignFraud(campaignId, signal);
    },
    [campaignId, refreshToken, tab]
  );

  const bumpFraudRefresh = useCoalescedBumpRefresh(bumpRefresh, fraudResource.fetching);

  const contextLoad = useCampaignEditorContextLoad(campaignId, tab === 'context');
  const integrationWorkspace = useCampaignIntegrationPanelWorkspace(campaignId);
  const fraudWorkspace = useCampaignFraudPanelWorkspace({
    campaignId,
    fraudConfig: fraudResource.data as CampaignFraudConfig | undefined,
    onSaved: bumpFraudRefresh,
  });
  const opsWorkspace = useCampaignOpsPanelWorkspace({ campaignId });

  return {
    tab,
    setTab,
    integration: {
      data: integrationResource.data,
      error: integrationResource.error,
      fetching: integrationResource.fetching,
      workspace: integrationWorkspace,
    },
    fraud: {
      data: fraudResource.data as CampaignFraudConfig | undefined,
      error: fraudResource.error,
      fetching: fraudResource.fetching,
      bumpRefresh: bumpFraudRefresh,
      workspace: fraudWorkspace,
    },
    ops: {
      workspace: opsWorkspace,
    },
    context: contextLoad,
  };
}
