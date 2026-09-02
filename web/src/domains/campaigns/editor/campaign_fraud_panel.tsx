import { useCallback, useEffect, useState } from 'react';

import {
  getCampaignFraud,
  patchCampaignFraud,
  previewCampaignFraud,
} from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignFraudConfig, CampaignFraudPreview } from '@/api/types';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from 'sonner';
import { useResource } from '@/api/use_resource';

function panelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}

function configToDraft(config: CampaignFraudConfig) {
  return {
    preset: '',
    fraud_threshold_pass: String(config.fraud_threshold_pass ?? ''),
    fraud_threshold_suspect: String(config.fraud_threshold_suspect ?? ''),
    fraud_threshold_ivt: String(config.fraud_threshold_ivt ?? ''),
    fraud_threshold_block: String(config.fraud_threshold_block ?? ''),
    silent_reject_enabled: config.silent_reject_enabled ?? false,
  };
}

export function CampaignFraudPanel({ campaignId }: { campaignId: string }) {
  const [draftPreset, setDraftPreset] = useState('');
  const [draftPass, setDraftPass] = useState('');
  const [draftSuspect, setDraftSuspect] = useState('');
  const [draftIvt, setDraftIvt] = useState('');
  const [draftBlock, setDraftBlock] = useState('');
  const [draftSilentReject, setDraftSilentReject] = useState(false);
  const [saving, setSaving] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [preview, setPreview] = useState<CampaignFraudPreview | undefined>();
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [previewError, setPreviewError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);

  const fraudResource = useResource(
    (signal) => getCampaignFraud(campaignId, signal),
    [campaignId, refreshToken],
  );

  useEffect(() => {
    if (!fraudResource.data) {
      return;
    }
    const draft = configToDraft(fraudResource.data);
    setDraftPreset(draft.preset);
    setDraftPass(draft.fraud_threshold_pass);
    setDraftSuspect(draft.fraud_threshold_suspect);
    setDraftIvt(draft.fraud_threshold_ivt);
    setDraftBlock(draft.fraud_threshold_block);
    setDraftSilentReject(draft.silent_reject_enabled);
    setPreview(undefined);
    setSaveSuccess(false);
  }, [fraudResource.data]);

  const parseOptionalInt = (raw: string): number | undefined => {
    const trimmed = raw.trim();
    if (!trimmed) {
      return undefined;
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (!Number.isFinite(parsed)) {
      throw new Error('Thresholds must be integers.');
    }
    return parsed;
  };

  const onSave = useCallback(async () => {
    setSaving(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await patchCampaignFraud(campaignId, {
        preset: draftPreset.trim() || undefined,
        fraud_threshold_pass: parseOptionalInt(draftPass),
        fraud_threshold_suspect: parseOptionalInt(draftSuspect),
        fraud_threshold_ivt: parseOptionalInt(draftIvt),
        fraud_threshold_block: parseOptionalInt(draftBlock),
        silent_reject_enabled: draftSilentReject,
      });
      setSaveSuccess(true);
      toast.success('Fraud config saved');
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [
    campaignId,
    draftBlock,
    draftIvt,
    draftPass,
    draftPreset,
    draftSilentReject,
    draftSuspect,
  ]);

  const onPreview = useCallback(async () => {
    setPreviewing(true);
    setPreviewError(undefined);
    setPreview(undefined);
    try {
      const result = await previewCampaignFraud(campaignId, {
        preset: draftPreset.trim() || undefined,
        fraud_threshold_pass: parseOptionalInt(draftPass),
        fraud_threshold_suspect: parseOptionalInt(draftSuspect),
        fraud_threshold_ivt: parseOptionalInt(draftIvt),
        fraud_threshold_block: parseOptionalInt(draftBlock),
      });
      setPreview(result);
    } catch (err) {
      setPreviewError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setPreviewing(false);
    }
  }, [campaignId, draftBlock, draftIvt, draftPass, draftPreset, draftSuspect]);

  return (
    <div className="grid gap-4">
      {fraudResource.error && !fraudResource.data
        ? panelError(fraudResource.error, 'Could not load fraud config')
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
        <Label htmlFor="fraud-silent-reject">Silent reject enabled</Label>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button disabled={saving || fraudResource.fetching} onClick={onSave} type="button">
          {saving ? 'Saving...' : 'Save fraud config'}
        </Button>
        <Button
          disabled={previewing || fraudResource.fetching}
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
      {saveError ? panelError(saveError, 'Could not save fraud config') : null}
      {previewError ? panelError(previewError, 'Could not preview fraud impact') : null}
      {preview ? (
        <section className="ui-filter-panel gap-2 text-sm">
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
