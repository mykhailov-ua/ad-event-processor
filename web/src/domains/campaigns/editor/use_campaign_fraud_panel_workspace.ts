// L3 fraud panel: local draft until PATCH; resets from fraudConfig prop when parent refetches.
// Toast after await patchCampaignFraud (EH-SI1); errors surface via saveError/previewError in panel.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import { patchCampaignFraud, previewCampaignFraud } from '@/api/campaigns_api';
import type { CampaignFraudConfig, CampaignFraudPreview } from '@/api/types';

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

type UseCampaignFraudPanelWorkspaceArgs = {
  campaignId: string;
  fraudConfig: CampaignFraudConfig | undefined;
  onSaved: () => void;
};

export function useCampaignFraudPanelWorkspace({
  campaignId,
  fraudConfig,
  onSaved,
}: UseCampaignFraudPanelWorkspaceArgs) {
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

  useEffect(() => {
    if (!fraudConfig) {
      return;
    }
    const draft = configToDraft(fraudConfig);
    setDraftPreset(draft.preset);
    setDraftPass(draft.fraud_threshold_pass);
    setDraftSuspect(draft.fraud_threshold_suspect);
    setDraftIvt(draft.fraud_threshold_ivt);
    setDraftBlock(draft.fraud_threshold_block);
    setDraftSilentReject(draft.silent_reject_enabled);
    setPreview(undefined);
    setSaveSuccess(false);
  }, [fraudConfig]);

  const parseOptionalInt = useCallback((raw: string): number | undefined => {
    const trimmed = raw.trim();
    if (!trimmed) {
      return undefined;
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (!Number.isFinite(parsed)) {
      throw new Error('Thresholds must be integers.');
    }
    return parsed;
  }, []);

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
      onSaved();
    } catch (err: unknown) {
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
    onSaved,
    parseOptionalInt,
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
    } catch (err: unknown) {
      setPreviewError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setPreviewing(false);
    }
  }, [campaignId, draftBlock, draftIvt, draftPass, draftPreset, draftSuspect, parseOptionalInt]);

  return {
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
  };
}

export type CampaignFraudPanelWorkspace = ReturnType<typeof useCampaignFraudPanelWorkspace>;
