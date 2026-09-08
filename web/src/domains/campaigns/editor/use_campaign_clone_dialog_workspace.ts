// campaign clone overlay: preview on open; POST clone when options confirmed.
import { useCallback, useEffect, useState } from 'react';

import {
  cloneCampaign,
  previewCampaignClone,
  type CloneCampaignOptions,
  type CloneCampaignPreview,
} from '@/api/campaigns_api';
import { isAbortError } from '@/api/client';
import {
  buildCloneRequestBody,
  DEFAULT_CLONE_OPTIONS,
} from '@/domains/campaigns/editor/campaign_clone_request';
import { newRandomUuid } from '@/lib/uuid';

type UseCampaignCloneDialogWorkspaceArgs = {
  campaignId: string | undefined;
  open: boolean;
  onCloned?: (newCampaignId: string) => void;
};

export function useCampaignCloneDialogWorkspace({
  campaignId,
  open,
  onCloned,
}: UseCampaignCloneDialogWorkspaceArgs) {
  const [nameSuffix, setNameSuffix] = useState(' (copy)');
  const [cloneOptions, setCloneOptions] = useState<CloneCampaignOptions>(DEFAULT_CLONE_OPTIONS);
  const [preview, setPreview] = useState<CloneCampaignPreview | undefined>();
  const [previewError, setPreviewError] = useState<Error | undefined>();
  const [previewing, setPreviewing] = useState(false);
  const [cloning, setCloning] = useState(false);
  const [cloneError, setCloneError] = useState<Error | undefined>();
  const [clonedId, setClonedId] = useState<string | undefined>();

  useEffect(() => {
    if (!open) {
      setPreview(undefined);
      setPreviewError(undefined);
      setCloneError(undefined);
      setClonedId(undefined);
      setNameSuffix(' (copy)');
      setCloneOptions(DEFAULT_CLONE_OPTIONS);
    }
  }, [open]);

  const onPreview = useCallback(() => {
    if (!campaignId) {
      return;
    }
    setPreviewing(true);
    setPreviewError(undefined);
    void previewCampaignClone(campaignId, buildCloneRequestBody(nameSuffix, cloneOptions))
      .then((result) => setPreview(result))
      .catch((err: unknown) => {
        if (!isAbortError(err)) {
          setPreviewError(err instanceof Error ? err : new Error(String(err)));
        }
      })
      .finally(() => setPreviewing(false));
  }, [campaignId, cloneOptions, nameSuffix]);

  const onClone = useCallback(() => {
    if (!campaignId) {
      return;
    }
    setCloning(true);
    setCloneError(undefined);
    void cloneCampaign(campaignId, buildCloneRequestBody(nameSuffix, cloneOptions), {
      idempotencyKey: newRandomUuid(),
    })
      .then((result) => {
        setClonedId(result.id);
        onCloned?.(result.id);
      })
      .catch((err: unknown) => {
        if (!isAbortError(err)) {
          setCloneError(err instanceof Error ? err : new Error(String(err)));
        }
      })
      .finally(() => setCloning(false));
  }, [campaignId, cloneOptions, nameSuffix, onCloned]);

  const onCloneOptionChange = useCallback((field: keyof CloneCampaignOptions, checked: boolean) => {
    setCloneOptions((current) => ({ ...current, [field]: checked }));
  }, []);

  return {
    nameSuffix,
    setNameSuffix,
    cloneOptions,
    onCloneOptionChange,
    preview,
    previewError,
    previewing,
    cloning,
    cloneError,
    clonedId,
    onPreview,
    onClone,
  };
}

export type CampaignCloneDialogWorkspace = ReturnType<typeof useCampaignCloneDialogWorkspace>;
