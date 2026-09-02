import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  checkCampaignPublish,
  cloneCampaign,
  exportCampaign,
  getCampaign,
  getCampaignDiff,
  patchCampaign,
  previewCampaignClone,
  previewCampaignMacros,
  publishCampaign,
  putCampaignOwner,
  validateCampaignPatch,
  type CampaignDiffResponse,
  type CloneCampaignOptions,
  type CloneCampaignPreview,
  type CloneCampaignRequest,
  type MacroPreviewResponse,
} from '@/api/campaigns_api';
import { isAbortError } from '@/api/client';
import { getFlow } from '@/api/flows_api';
import type {
  Campaign,
  CampaignPublishBlockedError,
  CampaignPublishCheck,
  CampaignValidateResponse,
} from '@/api/types';
import {
  buildCampaignPatchBody,
  CampaignEditor,
  campaignToFormState,
  type CampaignEditorFormState,
  type MacroPreviewFormState,
} from '@/domains/campaigns/editor/campaign_editor';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';

const DEFAULT_CLONE_OPTIONS: Required<CloneCampaignOptions> = {
  include_flow: true,
  include_postbacks: true,
  include_fraud: true,
  include_placement_blocks: true,
  reset_spend: false,
};

function buildCloneRequestBody(
  nameSuffix: string,
  options: CloneCampaignOptions,
): CloneCampaignRequest {
  const body: CloneCampaignRequest = { options };
  const suffix = nameSuffix.trim();
  if (suffix !== '') {
    body.name_suffix = suffix;
  }
  return body;
}

export function CampaignEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [form, setForm] = useState<CampaignEditorFormState | undefined>(undefined);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>(undefined);
  const [campaignSnapshot, setCampaignSnapshot] = useState<Campaign | undefined>(undefined);
  const [checking, setChecking] = useState(false);
  const [validating, setValidating] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [forcePublish, setForcePublish] = useState(false);
  const [publishCheck, setPublishCheck] = useState<CampaignPublishCheck | undefined>(undefined);
  const [validateResult, setValidateResult] = useState<
    CampaignValidateResponse | undefined
  >(undefined);
  const [publishBlocked, setPublishBlocked] = useState<
    CampaignPublishBlockedError | undefined
  >(undefined);
  const [publishSuccess, setPublishSuccess] = useState(false);
  const [publishCheckError, setPublishCheckError] = useState<Error | undefined>(undefined);
  const [validateError, setValidateError] = useState<Error | undefined>(undefined);
  const [publishError, setPublishError] = useState<Error | undefined>(undefined);
  const [macroPreviewForm, setMacroPreviewForm] = useState<MacroPreviewFormState>({
    sub1: '',
    country: '',
    click_id: '',
  });
  const [macroPreviewing, setMacroPreviewing] = useState(false);
  const [macroPreviewResult, setMacroPreviewResult] = useState<
    MacroPreviewResponse | undefined
  >(undefined);
  const [macroPreviewError, setMacroPreviewError] = useState<Error | undefined>(undefined);
  const [cloneNameSuffix, setCloneNameSuffix] = useState('');
  const [cloneOptions, setCloneOptions] = useState<CloneCampaignOptions>(DEFAULT_CLONE_OPTIONS);
  const [clonePreviewing, setClonePreviewing] = useState(false);
  const [clonePreview, setClonePreview] = useState<CloneCampaignPreview | undefined>(undefined);
  const [clonePreviewError, setClonePreviewError] = useState<Error | undefined>(undefined);
  const [cloning, setCloning] = useState(false);
  const [cloneSuccess, setCloneSuccess] = useState(false);
  const [clonedCampaignId, setClonedCampaignId] = useState<string | undefined>(undefined);
  const [cloneError, setCloneError] = useState<Error | undefined>(undefined);
  const [diffAgainstId, setDiffAgainstId] = useState('');
  const [comparingDiff, setComparingDiff] = useState(false);
  const [diffResult, setDiffResult] = useState<CampaignDiffResponse | undefined>(undefined);
  const [diffError, setDiffError] = useState<Error | undefined>(undefined);
  const [draftOwnerUserId, setDraftOwnerUserId] = useState('');
  const [transferringOwner, setTransferringOwner] = useState(false);
  const [ownerError, setOwnerError] = useState<Error | undefined>(undefined);
  const [ownerSuccess, setOwnerSuccess] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<Error | undefined>(undefined);
  const autoPublishCheckDone = useRef(false);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(new Error('Campaign id is required'));
      }
      return getCampaign(id, signal);
    },
    [id],
  );

  const flowId = (campaignSnapshot ?? data)?.flow_id?.trim() ?? '';

  const { data: flowData } = useResource(
    (signal) => {
      if (!flowId) {
        return Promise.resolve(undefined);
      }
      return getFlow(flowId, signal);
    },
    [flowId],
  );

  useEffect(() => {
    if (!data) {
      return;
    }
    setCampaignSnapshot(data);
    setForm(campaignToFormState(data));
    setSaveError(undefined);
  }, [data]);

  useEffect(() => {
    autoPublishCheckDone.current = false;
    setPublishCheck(undefined);
    setPublishCheckError(undefined);
  }, [id]);

  useEffect(() => {
    if (!id || !data || autoPublishCheckDone.current) {
      return;
    }
    if (data.status !== 'PAUSED') {
      return;
    }
    autoPublishCheckDone.current = true;

    setChecking(true);
    setPublishCheckError(undefined);

    void checkCampaignPublish(id)
      .then((result) => {
        setPublishCheck(result);
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setPublishCheckError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setChecking(false);
      });
  }, [data, id]);

  const onFieldChange = useCallback(
    <K extends keyof CampaignEditorFormState>(
      field: K,
      value: CampaignEditorFormState[K],
    ) => {
      setForm((prev) => {
        if (!prev) {
          return prev;
        }
        return { ...prev, [field]: value };
      });
    },
    [],
  );

  const onSave = useCallback(() => {
    if (!id || !campaignSnapshot || !form) {
      return;
    }

    const patchResult = buildCampaignPatchBody(campaignSnapshot, form);
    if (!patchResult.ok) {
      setSaveError(new Error(patchResult.error));
      return;
    }
    if (Object.keys(patchResult.body).length === 0) {
      return;
    }

    setSaving(true);
    setSaveError(undefined);

    void patchCampaign(id, patchResult.body)
      .then((updated) => {
        setCampaignSnapshot(updated);
        setForm(campaignToFormState(updated));
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setSaveError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setSaving(false);
      });
  }, [campaignSnapshot, form, id]);

  const onSaveAndClose = useCallback(() => {
    if (!id || !campaignSnapshot || !form) {
      return;
    }

    const patchResult = buildCampaignPatchBody(campaignSnapshot, form);
    if (!patchResult.ok) {
      setSaveError(new Error(patchResult.error));
      return;
    }
    if (Object.keys(patchResult.body).length === 0) {
      navigate('/campaigns');
      return;
    }

    setSaving(true);
    setSaveError(undefined);

    void patchCampaign(id, patchResult.body)
      .then((updated) => {
        setCampaignSnapshot(updated);
        setForm(campaignToFormState(updated));
        navigate('/campaigns');
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setSaveError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setSaving(false);
      });
  }, [campaignSnapshot, form, id, navigate]);

  const onCheckPublish = useCallback(() => {
    if (!id) {
      return;
    }

    setChecking(true);
    setPublishCheckError(undefined);
    setPublishCheck(undefined);

    void checkCampaignPublish(id)
      .then((result) => {
        setPublishCheck(result);
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setPublishCheckError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setChecking(false);
      });
  }, [id]);

  const onValidateChanges = useCallback(() => {
    if (!id || !campaignSnapshot || !form) {
      return;
    }

    const patchResult = buildCampaignPatchBody(campaignSnapshot, form);
    if (!patchResult.ok) {
      setValidateError(new Error(patchResult.error));
      setValidateResult(undefined);
      return;
    }
    if (Object.keys(patchResult.body).length === 0) {
      setValidateError(new Error('No unsaved changes to validate.'));
      setValidateResult(undefined);
      return;
    }

    setValidating(true);
    setValidateError(undefined);
    setValidateResult(undefined);

    void validateCampaignPatch(id, patchResult.body)
      .then((result) => {
        setValidateResult(result);
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setValidateError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setValidating(false);
      });
  }, [campaignSnapshot, form, id]);

  const onPublish = useCallback(() => {
    if (!id) {
      return;
    }

    setPublishing(true);
    setPublishError(undefined);
    setPublishBlocked(undefined);
    setPublishSuccess(false);

    void publishCampaign(id, { force: forcePublish })
      .then((result) => {
        if (result.status === 'published') {
          setPublishSuccess(true);
          toast.success('Campaign published');
          setCampaignSnapshot(result.campaign);
          setForm(campaignToFormState(result.campaign));
          return;
        }
        setPublishBlocked(result.error);
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setPublishError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setPublishing(false);
      });
  }, [forcePublish, id]);

  const onMacroPreviewFieldChange = useCallback(
    <K extends keyof MacroPreviewFormState>(
      field: K,
      value: MacroPreviewFormState[K],
    ) => {
      setMacroPreviewForm((prev) => ({ ...prev, [field]: value }));
    },
    [],
  );

  const onMacroPreview = useCallback(() => {
    if (!id) {
      return;
    }

    setMacroPreviewing(true);
    setMacroPreviewError(undefined);
    setMacroPreviewResult(undefined);

    const body: Parameters<typeof previewCampaignMacros>[1] = {};
    if (macroPreviewForm.sub1.trim() !== '') {
      body.sub1 = macroPreviewForm.sub1.trim();
    }
    if (macroPreviewForm.country.trim() !== '') {
      body.country = macroPreviewForm.country.trim();
    }
    if (macroPreviewForm.click_id.trim() !== '') {
      body.click_id = macroPreviewForm.click_id.trim();
    }

    void previewCampaignMacros(id, body)
      .then((result) => {
        setMacroPreviewResult(result);
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setMacroPreviewError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setMacroPreviewing(false);
      });
  }, [id, macroPreviewForm.click_id, macroPreviewForm.country, macroPreviewForm.sub1]);

  const onClonePreview = useCallback(() => {
    if (!id) {
      return;
    }

    setClonePreviewing(true);
    setClonePreviewError(undefined);
    setClonePreview(undefined);

    void previewCampaignClone(id, buildCloneRequestBody(cloneNameSuffix, cloneOptions))
      .then((result) => {
        setClonePreview(result);
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setClonePreviewError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setClonePreviewing(false);
      });
  }, [cloneNameSuffix, cloneOptions, id]);

  const onCloneExecute = useCallback(() => {
    if (!id) {
      return;
    }

    setCloning(true);
    setCloneError(undefined);
    setCloneSuccess(false);
    setClonedCampaignId(undefined);

    void cloneCampaign(id, buildCloneRequestBody(cloneNameSuffix, cloneOptions), {
      idempotencyKey: crypto.randomUUID(),
    })
      .then((result) => {
        setCloneSuccess(true);
        setClonedCampaignId(result.id);
        toast.success('Campaign clone created');
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setCloneError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setCloning(false);
      });
  }, [cloneNameSuffix, cloneOptions, id]);

  const onCloneOptionChange = useCallback(
    (field: keyof CloneCampaignOptions, value: boolean) => {
      setCloneOptions((prev) => ({ ...prev, [field]: value }));
    },
    [],
  );

  const onCompareDiff = useCallback(() => {
    if (!id) {
      return;
    }

    const against = diffAgainstId.trim();
    if (against === '') {
      setDiffError(new Error('Enter a campaign id to compare against.'));
      setDiffResult(undefined);
      return;
    }

    setComparingDiff(true);
    setDiffError(undefined);
    setDiffResult(undefined);

    void getCampaignDiff(id, against)
      .then((result) => {
        setDiffResult(result);
      })
      .catch((err: unknown) => {
        if (isAbortError(err)) {
          return;
        }
        setDiffError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setComparingDiff(false);
      });
  }, [diffAgainstId, id]);

  const onTransferOwner = useCallback(() => {
    if (!id) {
      return;
    }
    const userId = draftOwnerUserId.trim();
    if (!userId) {
      setOwnerError(new Error('Owner user ID is required.'));
      return;
    }
    setTransferringOwner(true);
    setOwnerError(undefined);
    setOwnerSuccess(false);
    void putCampaignOwner(id, { user_id: userId })
      .then(() => {
        setOwnerSuccess(true);
      })
      .catch((err: unknown) => {
        setOwnerError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setTransferringOwner(false);
      });
  }, [draftOwnerUserId, id]);

  const onExportCampaign = useCallback(() => {
    if (!id) {
      return;
    }
    setExporting(true);
    setExportError(undefined);
    void exportCampaign(id)
      .then((bundle) => {
        const blob = new Blob([JSON.stringify(bundle, null, 2)], {
          type: 'application/json',
        });
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `campaign-${id}.json`;
        anchor.click();
        URL.revokeObjectURL(url);
      })
      .catch((err: unknown) => {
        setExportError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setExporting(false);
      });
  }, [id]);

  const campaign = campaignSnapshot ?? data;
  const effectiveForm =
    form ?? (campaign ? campaignToFormState(campaign) : undefined);
  const flowPaths = useMemo(() => {
    const paths = flowData?.paths;
    return Array.isArray(paths) ? paths : undefined;
  }, [flowData?.paths]);

  useBreadcrumbSegmentLabel(id, campaign?.name);

  return (
    <CampaignEditor
      campaign={campaign}
      clickUrl={macroPreviewResult?.resolved_click_url}
      flowPaths={flowPaths}
      form={
        effectiveForm ?? {
          name: '',
          status: '',
          budget_limit: '',
          pacing_mode: '',
          flow_id: '',
          brand_id: '',
          ingress_param: '',
          ingress_scale: '',
          ingress_max_micro: '',
          ingress_policy: '',
          traffic_template_id: '',
          click_query_params_json: '{}',
        }
      }
      fetching={fetching}
      saving={saving}
      loadError={error}
      saveError={saveError}
      hasSnapshot={campaignSnapshot != null || data != null}
      onFieldChange={onFieldChange}
      onSave={onSave}
      onSaveAndClose={onSaveAndClose}
      checking={checking}
      validating={validating}
      publishing={publishing}
      forcePublish={forcePublish}
      publishCheck={publishCheck}
      validateResult={validateResult}
      publishBlocked={publishBlocked}
      publishSuccess={publishSuccess}
      publishCheckError={publishCheckError}
      validateError={validateError}
      publishError={publishError}
      onForcePublishChange={setForcePublish}
      onCheckPublish={onCheckPublish}
      onValidateChanges={onValidateChanges}
      onPublish={onPublish}
      macroPreviewForm={macroPreviewForm}
      onMacroPreviewFieldChange={onMacroPreviewFieldChange}
      macroPreviewing={macroPreviewing}
      macroPreviewResult={macroPreviewResult}
      macroPreviewError={macroPreviewError}
      onMacroPreview={onMacroPreview}
      cloneNameSuffix={cloneNameSuffix}
      onCloneNameSuffixChange={setCloneNameSuffix}
      cloneOptions={cloneOptions}
      onCloneOptionChange={onCloneOptionChange}
      clonePreviewing={clonePreviewing}
      clonePreview={clonePreview}
      clonePreviewError={clonePreviewError}
      onClonePreview={onClonePreview}
      cloning={cloning}
      cloneSuccess={cloneSuccess}
      clonedCampaignId={clonedCampaignId}
      cloneError={cloneError}
      onCloneExecute={onCloneExecute}
      diffAgainstId={diffAgainstId}
      onDiffAgainstIdChange={setDiffAgainstId}
      comparingDiff={comparingDiff}
      diffResult={diffResult}
      diffError={diffError}
      onCompareDiff={onCompareDiff}
      draftOwnerUserId={draftOwnerUserId}
      onDraftOwnerUserIdChange={setDraftOwnerUserId}
      transferringOwner={transferringOwner}
      ownerError={ownerError}
      ownerSuccess={ownerSuccess}
      onTransferOwner={onTransferOwner}
      exporting={exporting}
      exportError={exportError}
      onExportCampaign={onExportCampaign}
    />
  );
}
