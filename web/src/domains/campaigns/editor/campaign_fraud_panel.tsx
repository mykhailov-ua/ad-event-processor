import { FILTER_PANEL_SUMMARY_CLASS, DirectoryFilterForm, FilterField } from '@/shell/filter_panel';
import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { CampaignFraudConfig } from '@/api/types';
import type { CampaignFraudPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_fraud_panel_workspace';
import { FraudLimitsDocLink } from '@/domains/fraud/fraud_limits_doc_link';
import { displayTimestamp } from '@/lib/display';

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
    draftBehaviorFlags,
    setDraftBehaviorFlags,
    draftCanvasRetest,
    setDraftCanvasRetest,
    draftCgnatPolicy,
    setDraftCgnatPolicy,
    draftAcceptLangGeo,
    setDraftAcceptLangGeo,
    draftJsonSerialization,
    setDraftJsonSerialization,
    draftConversionRules,
    setConversionRule,
    draftCrossLayerAction,
    setDraftCrossLayerAction,
    draftCrossLayerThreshold,
    setDraftCrossLayerThreshold,
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
      <FraudLimitsDocLink />
      <p className="text-sm text-muted-foreground">
        ML fraud boost is applied from a Redis snapshot on the tracker. Batch scoring runs in{' '}
        <span className="font-mono text-xs">cmd/fraud-scorer</span>; there is no inline model call on{' '}
        <span className="font-mono text-xs">/track</span>.
      </p>
      {fraudConfig?.ml_boost_last_refreshed_at ? (
        <p className="text-sm text-muted-foreground">
          Last ML boost refresh:{' '}
          <span className="font-mono text-xs text-foreground">
            {displayTimestamp(fraudConfig.ml_boost_last_refreshed_at)}
          </span>
        </p>
      ) : null}
      {loadError && !fraudConfig
        ? campaignPanelError(loadError, 'Could not load fraud config')
        : null}

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField className="md:col-span-2" htmlFor="fraud-preset" label="Preset">
          <Input
            id="fraud-preset"
            value={draftPreset}
            onChange={(event) => setDraftPreset(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="fraud-pass" label="Pass threshold">
          <Input
            id="fraud-pass"
            inputMode="numeric"
            value={draftPass}
            onChange={(event) => setDraftPass(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="fraud-suspect" label="Suspect threshold">
          <Input
            id="fraud-suspect"
            inputMode="numeric"
            value={draftSuspect}
            onChange={(event) => setDraftSuspect(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="fraud-ivt" label="IVT threshold">
          <Input
            id="fraud-ivt"
            inputMode="numeric"
            value={draftIvt}
            onChange={(event) => setDraftIvt(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="fraud-block" label="Block threshold">
          <Input
            id="fraud-block"
            inputMode="numeric"
            value={draftBlock}
            onChange={(event) => setDraftBlock(event.target.value)}
          />
        </FilterField>
      </DirectoryFilterForm>

      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-2">
          <Checkbox
            checked={draftSilentReject}
            id="fraud-silent-reject"
            onCheckedChange={(checked) => setDraftSilentReject(checked === true)}
          />
          <Label htmlFor="fraud-silent-reject">Non-blocking fraud response enabled</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={draftCanvasRetest}
            id="fraud-canvas-retest"
            onCheckedChange={(checked) => setDraftCanvasRetest(checked === true)}
          />
          <Label htmlFor="fraud-canvas-retest">Canvas retest enabled</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={draftCgnatPolicy}
            id="fraud-cgnat-policy"
            onCheckedChange={(checked) => setDraftCgnatPolicy(checked === true)}
          />
          <Label htmlFor="fraud-cgnat-policy">CGNAT IP policy enabled</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={draftAcceptLangGeo}
            id="fraud-accept-lang-geo"
            onCheckedChange={(checked) => setDraftAcceptLangGeo(checked === true)}
          />
          <Label htmlFor="fraud-accept-lang-geo">Accept-Language geo check</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={draftJsonSerialization}
            id="fraud-json-serialization"
            onCheckedChange={(checked) => setDraftJsonSerialization(checked === true)}
          />
          <Label htmlFor="fraud-json-serialization">JSON serialization enabled</Label>
        </div>
      </div>

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField htmlFor="fraud-behavior-flags" label="Behavior flags (bitmask)">
          <Input
            id="fraud-behavior-flags"
            inputMode="numeric"
            value={draftBehaviorFlags}
            onChange={(event) => setDraftBehaviorFlags(event.target.value)}
          />
        </FilterField>
      </DirectoryFilterForm>

      <section className="grid gap-3 border border-border p-3">
        <h3 className="m-0 text-sm font-semibold text-foreground">Conversion reject rules</h3>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={draftConversionRules.enabled}
            id="conversion-reject-enabled"
            onCheckedChange={(checked) => setConversionRule('enabled', checked === true)}
          />
          <Label htmlFor="conversion-reject-enabled">Enable conversion reject rules</Label>
        </div>
        <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
          <FilterField htmlFor="conversion-min-ttc" label="Min TTC (ms)">
            <Input
              disabled={!draftConversionRules.enabled}
              id="conversion-min-ttc"
              inputMode="numeric"
              value={draftConversionRules.min_ttc_ms}
              onChange={(event) => setConversionRule('min_ttc_ms', event.target.value)}
            />
          </FilterField>
        </DirectoryFilterForm>
        <div className="flex flex-wrap gap-4">
          {(
            [
              ['reject_no_click', 'Reject missing click'],
              ['reject_low_ttc', 'Reject low TTC'],
              ['reject_duplicate', 'Reject duplicate'],
              ['reject_ip_drift', 'Reject IP drift'],
              ['reject_datacenter_ip', 'Reject datacenter IP'],
            ] as const
          ).map(([field, label]) => (
            <div key={field} className="flex items-center gap-2">
              <Checkbox
                checked={draftConversionRules[field]}
                disabled={!draftConversionRules.enabled}
                id={`conversion-${field}`}
                onCheckedChange={(checked) => setConversionRule(field, checked === true)}
              />
              <Label htmlFor={`conversion-${field}`}>{label}</Label>
            </div>
          ))}
        </div>
      </section>

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField htmlFor="fraud-cross-layer-action" label="Cross-layer desync action">
          <Select
            value={draftCrossLayerAction}
            onValueChange={(value) => setDraftCrossLayerAction(value)}
          >
            <SelectTrigger id="fraud-cross-layer-action">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="off">Off</SelectItem>
              <SelectItem value="boost">Boost only (default)</SelectItem>
              <SelectItem value="safe_page">Route to safe page</SelectItem>
              <SelectItem value="block">Block click</SelectItem>
            </SelectContent>
          </Select>
        </FilterField>
        <FilterField htmlFor="fraud-cross-layer-threshold" label="Desync layer threshold">
          <Input
            id="fraud-cross-layer-threshold"
            inputMode="numeric"
            value={draftCrossLayerThreshold}
            onChange={(event) => setDraftCrossLayerThreshold(event.target.value)}
          />
        </FilterField>
      </DirectoryFilterForm>
      <p className="text-xs text-muted-foreground">
        Cross-layer policy counts distinct wire/safe-page mismatch layers (TCP, TLS JA4, client hints,
        Sec-Fetch, H2). Residential IPs may still pass individual L2 signals; see fraud signal limits
        doc.
      </p>

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
