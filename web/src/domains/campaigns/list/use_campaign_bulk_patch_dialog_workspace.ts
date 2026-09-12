// Bulk patch overlay: POST /campaigns/bulk-patch for selected campaigns.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import type { CampaignBulkPatchResultRow } from '@/api/campaigns_types';
import {
  buildCampaignBulkPatchPayload,
  EMPTY_CAMPAIGN_BULK_PATCH_DRAFT,
  EMPTY_CAMPAIGN_BULK_PATCH_ENABLED,
  validateCampaignBulkPatchDraft,
  type CampaignBulkPatchDraft,
  type CampaignBulkPatchEnabled,
  type CampaignBulkPatchFieldKey,
} from '@/domains/campaigns/list/campaign_bulk_patch_build';
import { bulkPatchCampaigns } from '@/domains/campaigns/list/campaign_list_bulk_actions';
import { toError } from '@/lib/admin_error';

type UseCampaignBulkPatchDialogWorkspaceArgs = {
  campaignIds: string[];
  open: boolean;
  onPatched?: () => void;
};

export function useCampaignBulkPatchDialogWorkspace({
  campaignIds,
  open,
  onPatched,
}: UseCampaignBulkPatchDialogWorkspaceArgs) {
  const [draft, setDraft] = useState<CampaignBulkPatchDraft>(EMPTY_CAMPAIGN_BULK_PATCH_DRAFT);
  const [enabled, setEnabled] = useState<CampaignBulkPatchEnabled>(
    EMPTY_CAMPAIGN_BULK_PATCH_ENABLED
  );
  const [patching, setPatching] = useState(false);
  const [patchError, setPatchError] = useState<Error | undefined>();
  const [validationError, setValidationError] = useState<string | undefined>();
  const [results, setResults] = useState<CampaignBulkPatchResultRow[] | undefined>();

  useEffect(() => {
    if (!open) {
      setDraft(EMPTY_CAMPAIGN_BULK_PATCH_DRAFT);
      setEnabled(EMPTY_CAMPAIGN_BULK_PATCH_ENABLED);
      setPatchError(undefined);
      setValidationError(undefined);
      setResults(undefined);
    }
  }, [open]);

  const onDraftChange = useCallback((field: CampaignBulkPatchFieldKey, value: string) => {
    setDraft((current) => ({ ...current, [field]: value }));
    setValidationError(undefined);
  }, []);

  const onEnabledChange = useCallback((field: CampaignBulkPatchFieldKey, checked: boolean) => {
    setEnabled((current) => ({ ...current, [field]: checked }));
    setValidationError(undefined);
  }, []);

  const onApplyPatch = useCallback(async () => {
    if (patching || campaignIds.length === 0) {
      return;
    }
    const validation = validateCampaignBulkPatchDraft(draft, enabled);
    if (validation) {
      setValidationError(validation);
      return;
    }
    const patch = buildCampaignBulkPatchPayload(draft, enabled);
    if (!patch) {
      setValidationError('Enable at least one field to update');
      return;
    }

    setPatching(true);
    setPatchError(undefined);
    setValidationError(undefined);
    setResults(undefined);
    try {
      const result = await bulkPatchCampaigns(campaignIds, patch);
      const succeeded = result.succeeded;
      const failed = result.failed;
      setResults([
        ...succeeded.map((id) => ({ id, ok: true as const })),
        ...failed.map((row) => ({ id: row.id, ok: false as const, error_code: row.error })),
      ]);
      if (succeeded.length > 0 && failed.length === 0) {
        toast.success(`Updated ${succeeded.length} campaign(s)`);
        onPatched?.();
      } else if (succeeded.length > 0) {
        toast.warning(`Updated ${succeeded.length}; ${failed.length} failed`);
        onPatched?.();
      } else {
        toast.error('Bulk edit failed for all selected campaigns');
      }
    } catch (err: unknown) {
      setPatchError(toError(err));
    } finally {
      setPatching(false);
    }
  }, [campaignIds, draft, enabled, onPatched, patching]);

  return {
    draft,
    enabled,
    onDraftChange,
    onEnabledChange,
    patching,
    patchError,
    validationError,
    results,
    onApplyPatch,
  };
}

export type CampaignBulkPatchDialogWorkspace = ReturnType<
  typeof useCampaignBulkPatchDialogWorkspace
>;
