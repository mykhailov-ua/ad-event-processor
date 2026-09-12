// campaigns create lane: draft defaults on dialog open / submit; toast after createSelfServeCampaign 2xx; refreshList coalesced by parent.
import { useCallback, useEffect, useRef, type Dispatch, type SetStateAction } from 'react';
import { toast } from 'sonner';

import { createSelfServeCampaign } from '@/api/selfserve_api';
import { invalidateCampaignListResponseCache } from '@/domains/campaigns/list/campaign_list_response_cache';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { toError, userErrorMessage } from '@/lib/admin_error';
import {
  actionGuardError,
  requirePositiveInteger,
  toastValidationError,
} from '@/lib/admin_validation_error';
import { newRandomUuid } from '@/lib/uuid';

export type UseCampaignsPageMutationsArgs = {
  customerId: string | undefined;
  appliedCustomerId: string;
  templates: Array<{ id?: string }>;
  templatesFetching: boolean;
  draftTemplateId: string;
  setDraftTemplateId: (value: string) => void;
  draftCreateCustomerId: string;
  draftCreateName: string;
  setDraftCreateName: (value: string) => void;
  draftBudgetLimitMicro: string;
  setDraftBudgetLimitMicro: (value: string) => void;
  setCreating: (value: boolean) => void;
  setActionError: (error: Error | undefined) => void;
  setTemplatesRefreshToken: Dispatch<SetStateAction<number>>;
  setCreateSectionOpen: (open: boolean) => void;
  createSectionOpen: boolean;
  refreshList: () => void;
};

function resolveCreateTemplateId(
  draftTemplateId: string,
  templates: Array<{ id?: string }>
): string {
  if (draftTemplateId && templates.some((template) => template.id === draftTemplateId)) {
    return draftTemplateId;
  }
  return templates[0]?.id ?? '';
}

export function useCampaignsPageMutations({
  customerId,
  appliedCustomerId,
  templates,
  templatesFetching,
  draftTemplateId,
  setDraftTemplateId,
  draftCreateCustomerId,
  draftCreateName,
  setDraftCreateName,
  draftBudgetLimitMicro,
  setDraftBudgetLimitMicro,
  setCreating,
  setActionError,
  setTemplatesRefreshToken,
  setCreateSectionOpen,
  createSectionOpen,
  refreshList,
}: UseCampaignsPageMutationsArgs) {
  const createIdempotencyKeyRef = useRef<string | null>(null);

  useEffect(() => {
    if (createSectionOpen) {
      createIdempotencyKeyRef.current = newRandomUuid();
    } else {
      createIdempotencyKeyRef.current = null;
    }
  }, [createSectionOpen]);

  const onLoadTemplates = useCoalescedCallback(
    () => {
      setTemplatesRefreshToken((value) => value + 1);
    },
    { inFlightGuard: true, inFlight: templatesFetching }
  );

  const onCreateCampaign = useCallback(async () => {
    const effectiveCustomerId =
      draftCreateCustomerId.trim() || appliedCustomerId || customerId || '';
    const effectiveTemplateId = resolveCreateTemplateId(draftTemplateId, templates);
    if (!effectiveCustomerId) {
      const err = actionGuardError('Select a customer group before creating a campaign');
      setActionError(err);
      toastValidationError(err);
      return;
    }
    if (!effectiveTemplateId) {
      const err = actionGuardError('Select a template before creating a campaign');
      setActionError(err);
      toastValidationError(err);
      return;
    }

    const budgetRaw = draftBudgetLimitMicro.trim();
    let budgetLimitMicro: number | undefined;
    if (budgetRaw) {
      const parsed = requirePositiveInteger(budgetRaw, 'Budget', 'budget_limit_micro');
      if (!parsed.ok) {
        setActionError(parsed.error);
        return;
      }
      budgetLimitMicro = parsed.value;
    }

    if (!createIdempotencyKeyRef.current) {
      createIdempotencyKeyRef.current = newRandomUuid();
    }

    setCreating(true);
    setActionError(undefined);
    try {
      await createSelfServeCampaign(
        {
          customer_id: effectiveCustomerId,
          template_id: effectiveTemplateId,
          name: draftCreateName.trim() || undefined,
          budget_limit_micro: budgetLimitMicro,
        },
        { idempotencyKey: createIdempotencyKeyRef.current }
      );
      createIdempotencyKeyRef.current = null;
      setDraftCreateName('');
      setDraftBudgetLimitMicro('');
      setDraftTemplateId('');
      setCreateSectionOpen(false);
      toast.success('Campaign created');
      invalidateCampaignListResponseCache();
      refreshList();
    } catch (err: unknown) {
      const nextError = toError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
    } finally {
      setCreating(false);
    }
  }, [
    appliedCustomerId,
    customerId,
    draftBudgetLimitMicro,
    draftCreateCustomerId,
    draftCreateName,
    draftTemplateId,
    refreshList,
    setActionError,
    setCreateSectionOpen,
    setCreating,
    setDraftBudgetLimitMicro,
    setDraftCreateName,
    setDraftTemplateId,
    templates,
  ]);

  return {
    onLoadTemplates,
    onCreateCampaign,
  };
}
