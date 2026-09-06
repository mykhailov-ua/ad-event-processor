import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { formatCampaignJsonKey } from '@/domains/campaigns/editor/campaign_json_labels';
import type { useCampaignEditorContextLoad } from '@/domains/campaigns/editor/use_campaign_editor_context_load';

export function CampaignEditorContextPanel({
  campaignId: _campaignId,
  context,
}: {
  campaignId: string;
  context: ReturnType<typeof useCampaignEditorContextLoad>;
}) {
  const {
    loadKey,
    geoExpand,
    fraudPreview,
    geoResource,
    fraudResource,
    shellResource,
    busy,
    onLoadGeo,
    onLoadFraud,
    onLoadShell,
    onGeoExpandChange,
    onFraudPreviewChange,
  } = context;

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-2">
          <Checkbox
            checked={geoExpand}
            disabled={busy}
            id="editor-context-geo-expand"
            onCheckedChange={(checked) => onGeoExpandChange(checked === true)}
          />
          <Label htmlFor="editor-context-geo-expand">Expand geo rows</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={fraudPreview}
            disabled={busy}
            id="editor-context-fraud-preview"
            onCheckedChange={(checked) => onFraudPreviewChange(checked === true)}
          />
          <Label htmlFor="editor-context-fraud-preview">Fraud preview query</Label>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button disabled={busy} onClick={onLoadGeo} type="button" variant="outline">
          {loadKey === 'geo' && geoResource.fetching ? 'Loading...' : 'Geo summary'}
        </Button>
        <Button disabled={busy} onClick={onLoadFraud} type="button" variant="outline">
          {loadKey === 'fraud' && fraudResource.fetching ? 'Loading...' : 'Fraud editor'}
        </Button>
        <Button disabled={busy} onClick={onLoadShell} type="button" variant="outline">
          {loadKey === 'shell' && shellResource.fetching ? 'Loading...' : 'Editor shell'}
        </Button>
      </div>

      {loadKey === 'geo' && geoResource.error
        ? campaignPanelError(geoResource.error, 'Could not load geo summary')
        : null}
      {loadKey === 'fraud' && fraudResource.error
        ? campaignPanelError(fraudResource.error, 'Could not load fraud editor summary')
        : null}
      {loadKey === 'shell' && shellResource.error
        ? campaignPanelError(shellResource.error, 'Could not load editor shell')
        : null}

      {loadKey === 'geo' && geoResource.data ? (
        <JsonPayloadView
          formatColumn={formatCampaignJsonKey}
          formatKey={formatCampaignJsonKey}
          payload={geoResource.data}
        />
      ) : null}
      {loadKey === 'fraud' && fraudResource.data ? (
        <JsonPayloadView
          formatColumn={formatCampaignJsonKey}
          formatKey={formatCampaignJsonKey}
          payload={fraudResource.data}
        />
      ) : null}
      {loadKey === 'shell' && shellResource.data ? (
        <JsonPayloadView
          formatColumn={formatCampaignJsonKey}
          formatKey={formatCampaignJsonKey}
          payload={shellResource.data}
        />
      ) : null}
    </div>
  );
}
