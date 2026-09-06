import { Button } from '@/components/ui/button';
import { EDITOR_MAIN_COLUMN_CLASS } from '@/shell/filter_panel';
import { CampaignEditorContextPanel } from '@/domains/campaigns/editor/campaign_editor_context_panel';
import { CampaignFraudPanel } from '@/domains/campaigns/editor/campaign_fraud_panel';
import { CampaignIntegrationPanel } from '@/domains/campaigns/editor/campaign_integration_panel';
import { CampaignOpsPanel } from '@/domains/campaigns/editor/campaign_ops_panel';
import { useCampaignEditorToolsLoad } from '@/domains/campaigns/editor/use_campaign_editor_tools_load';
import { cn } from '@/lib/utils';

export type CampaignEditorToolsTab = 'integration' | 'fraud' | 'ops' | 'context';

const TOOL_TABS: { id: CampaignEditorToolsTab; label: string }[] = [
  { id: 'integration', label: 'Integration' },
  { id: 'fraud', label: 'Fraud' },
  { id: 'ops', label: 'Ops' },
  { id: 'context', label: 'Editor context' },
];

export function CampaignEditorTools({ campaignId }: { campaignId: string }) {
  const tools = useCampaignEditorToolsLoad(campaignId);

  return (
    <section className={cn(EDITOR_MAIN_COLUMN_CLASS, 'gap-4')}>
      <h2 className="text-sm font-semibold text-foreground">Campaign tools</h2>
      <div className="flex flex-wrap gap-2">
        {TOOL_TABS.map((item) => (
          <Button
            key={item.id}
            className="focus-visible:ring-0 focus-visible:ring-offset-0"
            type="button"
            variant={tools.tab === item.id ? 'default' : 'outline'}
            onClick={() => tools.setTab(item.id)}
          >
            {item.label}
          </Button>
        ))}
      </div>
      {tools.tab === 'integration' ? (
        <CampaignIntegrationPanel
          loadError={tools.integration.error}
          panel={tools.integration.data}
          panelFetching={tools.integration.fetching}
          workspace={tools.integration.workspace}
        />
      ) : null}
      {tools.tab === 'fraud' ? (
        <CampaignFraudPanel
          fetching={tools.fraud.fetching}
          fraudConfig={tools.fraud.data}
          loadError={tools.fraud.error}
          workspace={tools.fraud.workspace}
        />
      ) : null}
      {tools.tab === 'ops' ? <CampaignOpsPanel workspace={tools.ops.workspace} /> : null}
      {tools.tab === 'context' ? (
        <CampaignEditorContextPanel campaignId={campaignId} context={tools.context} />
      ) : null}
    </section>
  );
}
