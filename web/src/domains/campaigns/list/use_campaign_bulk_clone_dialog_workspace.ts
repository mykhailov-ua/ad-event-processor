// bulk clone overlay: POST bulk-clone for multi-selected campaigns.
import { useCallback, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';

import { bulkCloneCampaigns } from '@/api/campaigns_api';
import type { BulkCloneCampaignResultRow, CloneCampaignOptions } from '@/api/campaigns_types';
import {
  buildCloneRequestBody,
  DEFAULT_CLONE_OPTIONS,
} from '@/domains/campaigns/editor/campaign_clone_request';
import { toError } from '@/lib/admin_error';
import { newRandomUuid } from '@/lib/uuid';

type UseCampaignBulkCloneDialogWorkspaceArgs = {
  sourceCampaignIds: string[];
  customerId?: string;
  open: boolean;
  onCloned?: () => void;
};

export function useCampaignBulkCloneDialogWorkspace({
  sourceCampaignIds,
  customerId,
  open,
  onCloned,
}: UseCampaignBulkCloneDialogWorkspaceArgs) {
  const [nameSuffix, setNameSuffix] = useState(' (copy)');
  const [cloneOptions, setCloneOptions] = useState<CloneCampaignOptions>(DEFAULT_CLONE_OPTIONS);
  const [cloning, setCloning] = useState(false);
  const [cloneError, setCloneError] = useState<Error | undefined>();
  const [results, setResults] = useState<BulkCloneCampaignResultRow[] | undefined>();
  const idempotencyKeyRef = useRef<string | null>(null);

  useEffect(() => {
    if (!open) {
      idempotencyKeyRef.current = null;
      setCloneError(undefined);
      setResults(undefined);
      setNameSuffix(' (copy)');
      setCloneOptions(DEFAULT_CLONE_OPTIONS);
    } else if (!idempotencyKeyRef.current) {
      idempotencyKeyRef.current = newRandomUuid();
    }
  }, [open]);

  const onCloneOptionChange = useCallback((field: keyof CloneCampaignOptions, checked: boolean) => {
    setCloneOptions((current) => ({ ...current, [field]: checked }));
  }, []);

  const onBulkClone = useCallback(async () => {
    if (cloning || sourceCampaignIds.length === 0) {
      return;
    }
    if (!idempotencyKeyRef.current) {
      idempotencyKeyRef.current = newRandomUuid();
    }

    setCloning(true);
    setCloneError(undefined);
    setResults(undefined);
    try {
      const cloneBody = buildCloneRequestBody(nameSuffix, cloneOptions);
      const response = await bulkCloneCampaigns(
        {
          source_campaign_ids: sourceCampaignIds,
          customer_id: customerId?.trim() || undefined,
          name_prefix: cloneBody.name_prefix,
          name_suffix: cloneBody.name_suffix,
          options: cloneBody.options,
        },
        { idempotencyKey: idempotencyKeyRef.current }
      );
      idempotencyKeyRef.current = null;
      setResults(response.results);
      const { succeeded, failed } = summarizeBulkCloneResults(response.results);
      if (succeeded.length > 0 && failed.length === 0) {
        toast.success(`Cloned ${succeeded.length} campaign(s)`);
        onCloned?.();
      } else if (succeeded.length > 0) {
        toast.warning(`Cloned ${succeeded.length}; ${failed.length} failed`);
        onCloned?.();
      } else {
        toast.error('Bulk clone failed for all selected campaigns');
      }
    } catch (err: unknown) {
      setCloneError(toError(err));
    } finally {
      setCloning(false);
    }
  }, [cloneOptions, cloning, customerId, nameSuffix, onCloned, sourceCampaignIds]);

  return {
    nameSuffix,
    setNameSuffix,
    cloneOptions,
    onCloneOptionChange,
    cloning,
    cloneError,
    results,
    onBulkClone,
  };
}

function summarizeBulkCloneResults(results: BulkCloneCampaignResultRow[]) {
  const succeeded: BulkCloneCampaignResultRow[] = [];
  const failed: BulkCloneCampaignResultRow[] = [];
  for (const row of results) {
    if (row.ok) {
      succeeded.push(row);
    } else {
      failed.push(row);
    }
  }
  return { succeeded, failed };
}

export type CampaignBulkCloneDialogWorkspace = ReturnType<
  typeof useCampaignBulkCloneDialogWorkspace
>;
