// campaign clone overlay: preview on open; POST clone when options confirmed.
import { useCallback, useEffect, useRef, useState } from 'react';

import {
  cloneCampaign,
  previewCampaignClone,
  type CloneCampaignOptions,
  type CloneCampaignPreview,
} from '@/api/campaigns_api';
import { isAbortError } from '@/api/client';
import {
  buildCloneRequestBody,
  cloneRequestError,
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
  const previewAbortRef = useRef<AbortController | null>(null);
  const idempotencyKeyRef = useRef<string | null>(null);

  useEffect(() => {
    if (!open) {
      previewAbortRef.current?.abort();
      previewAbortRef.current = null;
      idempotencyKeyRef.current = null;
      setPreview(undefined);
      setPreviewError(undefined);
      setCloneError(undefined);
      setClonedId(undefined);
      setNameSuffix(' (copy)');
      setCloneOptions(DEFAULT_CLONE_OPTIONS);
    } else if (!idempotencyKeyRef.current) {
      idempotencyKeyRef.current = newRandomUuid();
    }
  }, [open]);

  useEffect(() => {
    return () => {
      previewAbortRef.current?.abort();
      previewAbortRef.current = null;
    };
  }, [campaignId, cloneOptions, nameSuffix, open]);

  const onPreview = useCallback(async () => {
    if (!campaignId) {
      return;
    }

    previewAbortRef.current?.abort();
    const controller = new AbortController();
    previewAbortRef.current = controller;

    setPreviewing(true);
    setPreviewError(undefined);

    try {
      const result = await previewCampaignClone(
        campaignId,
        buildCloneRequestBody(nameSuffix, cloneOptions),
        controller.signal
      );
      setPreview(result);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setPreviewError(cloneRequestError(err));
    } finally {
      if (previewAbortRef.current === controller) {
        previewAbortRef.current = null;
      }
      if (!controller.signal.aborted) {
        setPreviewing(false);
      }
    }
  }, [campaignId, cloneOptions, nameSuffix]);

  const onClone = useCallback(async () => {
    if (!campaignId) {
      return;
    }
    if (!idempotencyKeyRef.current) {
      idempotencyKeyRef.current = newRandomUuid();
    }

    setCloning(true);
    setCloneError(undefined);

    try {
      const result = await cloneCampaign(campaignId, buildCloneRequestBody(nameSuffix, cloneOptions), {
        idempotencyKey: idempotencyKeyRef.current,
      });
      idempotencyKeyRef.current = null;
      setClonedId(result.id);
      onCloned?.(result.id);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setCloneError(cloneRequestError(err));
    } finally {
      setCloning(false);
    }
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
