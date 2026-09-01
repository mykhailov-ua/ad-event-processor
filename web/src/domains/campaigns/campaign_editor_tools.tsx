import { useState } from 'react';

import { Button } from '@/components/ui/button';
import { CampaignEditorContextPanel } from '@/domains/campaigns/campaign_editor_context_panel';
import { CampaignFraudPanel } from '@/domains/campaigns/campaign_fraud_panel';
import { CampaignIntegrationPanel } from '@/domains/campaigns/campaign_integration_panel';
import { CampaignOpsPanel } from '@/domains/campaigns/campaign_ops_panel';

export type CampaignEditorToolsTab = 'integration' | 'fraud' | 'ops' | 'context';

const TOOL_TABS: { id: CampaignEditorToolsTab; label: string }[] = [
  { id: 'integration', label: 'Integration' },
  { id: 'fraud', label: 'Fraud' },
  { id: 'ops', label: 'Ops' },
  { id: 'context', label: 'Editor context' },
];

export function CampaignEditorTools({ campaignId }: { campaignId: string }) {
  const [tab, setTab] = useState<CampaignEditorToolsTab>('integration');

  return (
    <section className="ui-filter-panel">
      <h2 className="text-base font-semibold">Campaign tools</h2>
      <div className="flex flex-wrap gap-2">
        {TOOL_TABS.map((item) => (
          <Button
            key={item.id}
            type="button"
            variant={tab === item.id ? 'default' : 'outline'}
            onClick={() => setTab(item.id)}
          >
            {item.label}
          </Button>
        ))}
      </div>
      {tab === 'integration' ? <CampaignIntegrationPanel campaignId={campaignId} /> : null}
      {tab === 'fraud' ? <CampaignFraudPanel campaignId={campaignId} /> : null}
      {tab === 'ops' ? <CampaignOpsPanel campaignId={campaignId} /> : null}
      {tab === 'context' ? <CampaignEditorContextPanel campaignId={campaignId} /> : null}
    </section>
  );
}
