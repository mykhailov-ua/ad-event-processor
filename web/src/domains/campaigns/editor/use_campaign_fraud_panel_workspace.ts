// L3 fraud panel: local draft until PATCH; resets from fraudConfig prop when parent refetches.
// Toast after await patchCampaignFraud (EH-SI1); errors surface via saveError/previewError in panel.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import { patchCampaignFraud, previewCampaignFraud } from '@/api/campaigns_api';
import type {
  CampaignFraudConfig,
  CampaignFraudPreview,
  ConversionRejectRules,
  PatchCampaignFraudRequest,
} from '@/api/types';

type ConversionRejectDraft = {
  enabled: boolean;
  min_ttc_ms: string;
  reject_no_click: boolean;
  reject_low_ttc: boolean;
  reject_duplicate: boolean;
  reject_ip_drift: boolean;
  reject_datacenter_ip: boolean;
};

function conversionRulesToDraft(rules?: ConversionRejectRules): ConversionRejectDraft {
  return {
    enabled: rules?.enabled ?? false,
    min_ttc_ms: rules?.min_ttc_ms != null ? String(rules.min_ttc_ms) : '',
    reject_no_click: rules?.reject_no_click ?? false,
    reject_low_ttc: rules?.reject_low_ttc ?? false,
    reject_duplicate: rules?.reject_duplicate ?? false,
    reject_ip_drift: rules?.reject_ip_drift ?? false,
    reject_datacenter_ip: rules?.reject_datacenter_ip ?? false,
  };
}

function draftToConversionRules(draft: ConversionRejectDraft): ConversionRejectRules {
  const minTtc = draft.min_ttc_ms.trim();
  return {
    enabled: draft.enabled,
    min_ttc_ms: minTtc ? Number.parseInt(minTtc, 10) : undefined,
    reject_no_click: draft.reject_no_click,
    reject_low_ttc: draft.reject_low_ttc,
    reject_duplicate: draft.reject_duplicate,
    reject_ip_drift: draft.reject_ip_drift,
    reject_datacenter_ip: draft.reject_datacenter_ip,
  };
}

function configToDraft(config: CampaignFraudConfig) {
  return {
    preset: '',
    fraud_threshold_pass: String(config.fraud_threshold_pass ?? ''),
    fraud_threshold_suspect: String(config.fraud_threshold_suspect ?? ''),
    fraud_threshold_ivt: String(config.fraud_threshold_ivt ?? ''),
    fraud_threshold_block: String(config.fraud_threshold_block ?? ''),
    silent_reject_enabled: config.silent_reject_enabled ?? false,
    behavior_flags: String(config.behavior_flags ?? ''),
    canvas_retest_enabled: config.canvas_retest_enabled ?? false,
    cgnat_ip_policy_enabled: config.cgnat_ip_policy_enabled ?? false,
    accept_lang_geo_enabled: config.accept_lang_geo_enabled ?? false,
    json_serialization_enabled: config.json_serialization_enabled ?? false,
    conversion_reject_rules: conversionRulesToDraft(config.conversion_reject_rules),
    cross_layer_desync_action: config.cross_layer_desync_action ?? 'boost',
    cross_layer_desync_threshold: String(config.cross_layer_desync_threshold ?? 3),
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
  const [draftBehaviorFlags, setDraftBehaviorFlags] = useState('');
  const [draftCanvasRetest, setDraftCanvasRetest] = useState(false);
  const [draftCgnatPolicy, setDraftCgnatPolicy] = useState(false);
  const [draftAcceptLangGeo, setDraftAcceptLangGeo] = useState(false);
  const [draftJsonSerialization, setDraftJsonSerialization] = useState(false);
  const [draftConversionRules, setDraftConversionRules] = useState<ConversionRejectDraft>(
    conversionRulesToDraft()
  );
  const [draftCrossLayerAction, setDraftCrossLayerAction] = useState('boost');
  const [draftCrossLayerThreshold, setDraftCrossLayerThreshold] = useState('3');
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
    setDraftBehaviorFlags(draft.behavior_flags);
    setDraftCanvasRetest(draft.canvas_retest_enabled);
    setDraftCgnatPolicy(draft.cgnat_ip_policy_enabled);
    setDraftAcceptLangGeo(draft.accept_lang_geo_enabled);
    setDraftJsonSerialization(draft.json_serialization_enabled);
    setDraftConversionRules(draft.conversion_reject_rules);
    setDraftCrossLayerAction(draft.cross_layer_desync_action);
    setDraftCrossLayerThreshold(draft.cross_layer_desync_threshold);
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

  const buildPatchBody = useCallback(() => {
    return {
      preset: draftPreset.trim() || undefined,
      fraud_threshold_pass: parseOptionalInt(draftPass),
      fraud_threshold_suspect: parseOptionalInt(draftSuspect),
      fraud_threshold_ivt: parseOptionalInt(draftIvt),
      fraud_threshold_block: parseOptionalInt(draftBlock),
      silent_reject_enabled: draftSilentReject,
      behavior_flags: parseOptionalInt(draftBehaviorFlags),
      canvas_retest_enabled: draftCanvasRetest,
      cgnat_ip_policy_enabled: draftCgnatPolicy,
      accept_lang_geo_enabled: draftAcceptLangGeo,
      json_serialization_enabled: draftJsonSerialization,
      conversion_reject_rules: draftToConversionRules(draftConversionRules),
      cross_layer_desync_action:
        draftCrossLayerAction as PatchCampaignFraudRequest['cross_layer_desync_action'],
      cross_layer_desync_threshold: parseOptionalInt(draftCrossLayerThreshold),
    };
  }, [
    draftAcceptLangGeo,
    draftBehaviorFlags,
    draftBlock,
    draftCanvasRetest,
    draftCgnatPolicy,
    draftConversionRules,
    draftCrossLayerAction,
    draftCrossLayerThreshold,
    draftIvt,
    draftJsonSerialization,
    draftPass,
    draftPreset,
    draftSilentReject,
    draftSuspect,
    parseOptionalInt,
  ]);

  const onSave = useCallback(async () => {
    setSaving(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await patchCampaignFraud(campaignId, buildPatchBody());
      setSaveSuccess(true);
      toast.success('Fraud config saved');
      onSaved();
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [buildPatchBody, campaignId, onSaved]);

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

  const setConversionRule = useCallback(
    <K extends keyof ConversionRejectDraft>(field: K, value: ConversionRejectDraft[K]) => {
      setDraftConversionRules((current) => ({ ...current, [field]: value }));
    },
    []
  );

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
  };
}

export type CampaignFraudPanelWorkspace = ReturnType<typeof useCampaignFraudPanelWorkspace>;
