import { FILTER_PANEL_SUMMARY_CLASS } from '@/shell/filter_panel';
import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { CampaignFraudConfig } from '@/api/types';
import type { CampaignFraudPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_fraud_panel_workspace';

export type CampaignFraudPanelProps = {
  fetching: boolean;
  fraudConfig: CampaignFraudConfig | undefined;
  loadError: Error | undefined;
  workspace: CampaignFraudPanelWorkspace;
};

export function CampaignFraudPanel({
  fetching,
  fraudConfig,
  loadError,
  workspace,
}: CampaignFraudPanelProps) {
  const {
    draftPreset,
    setDraftPreset,
    draftPass,
    setDraftPass,
    draftSuspect,
    setDraftSuspect,
    draftIvt,
    setDraftIvt,
    draftBlock,
    setDraftBlock,
    draftSilentReject,
    setDraftSilentReject,
    saving,
    previewing,
    preview,
    saveError,
    previewError,
    saveSuccess,
    onSave,
    onPreview,
  } = workspace;

  return (
    <div className="grid gap-4">
      {loadError && !fraudConfig
        ? campaignPanelError(loadError, 'Could not load fraud config')
        : null}

      <div className="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] items-end gap-4">
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="fraud-preset">Preset</Label>
          <Input
            id="fraud-preset"
            value={draftPreset}
            onChange={(event) => setDraftPreset(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="fraud-pass">Pass threshold</Label>
          <Input
            id="fraud-pass"
            inputMode="numeric"
            value={draftPass}
            onChange={(event) => setDraftPass(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="fraud-suspect">Suspect threshold</Label>
          <Input
            id="fraud-suspect"
            inputMode="numeric"
            value={draftSuspect}
            onChange={(event) => setDraftSuspect(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="fraud-ivt">IVT threshold</Label>
          <Input
            id="fraud-ivt"
            inputMode="numeric"
            value={draftIvt}
            onChange={(event) => setDraftIvt(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="fraud-block">Block threshold</Label>
          <Input
            id="fraud-block"
            inputMode="numeric"
            value={draftBlock}
            onChange={(event) => setDraftBlock(event.target.value)}
          />
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Checkbox
          checked={draftSilentReject}
          id="fraud-silent-reject"
          onCheckedChange={(checked) => setDraftSilentReject(checked === true)}
        />
        <Label htmlFor="fraud-silent-reject">Non-blocking fraud response enabled</Label>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button disabled={saving || fetching} onClick={onSave} type="button">
          {saving ? 'Saving...' : 'Save fraud config'}
        </Button>
        <Button
          disabled={previewing || fetching}
          onClick={onPreview}
          type="button"
          variant="secondary"
        >
          {previewing ? 'Previewing...' : 'Preview impact'}
        </Button>
      </div>

      {saveSuccess ? (
        <p className="text-sm text-muted-foreground" role="status">
          Fraud config saved.
        </p>
      ) : null}
      {saveError ? campaignPanelError(saveError, 'Could not save fraud config') : null}
      {previewError ? campaignPanelError(previewError, 'Could not preview fraud impact') : null}
      {preview ? (
        <section className={FILTER_PANEL_SUMMARY_CLASS}>
          <p>
            Affected IPs (7d): <strong>{preview.affected_ips_7d ?? 0}</strong>
          </p>
          <p>
            Sample size: <strong>{preview.sample_size ?? 0}</strong>
          </p>
          <p className="text-muted-foreground">{preview.disclaimer}</p>
        </section>
      ) : null}
    </div>
  );
}
