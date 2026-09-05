import { useCallback, useEffect, type Dispatch, type SetStateAction } from 'react';
import { toast } from 'sonner';

import { createSelfServeCampaign } from '@/api/selfserve_api';

export type UseCampaignsPageMutationsArgs = {
  customerId: string | undefined;
  appliedCustomerId: string;
  createSectionOpen: boolean;
  setCreateSectionOpen: (open: boolean) => void;
  templates: Array<{ id?: string }>;
  draftTemplateId: string;
  setDraftTemplateId: (value: string) => void;
  draftCreateCustomerId: string;
  setDraftCreateCustomerId: (value: string) => void;
  draftCreateName: string;
  setDraftCreateName: (value: string) => void;
  draftBudgetLimitMicro: string;
  setDraftBudgetLimitMicro: (value: string) => void;
  setCreating: (value: boolean) => void;
  setActionError: (error: Error | undefined) => void;
  setTemplatesRefreshToken: Dispatch<SetStateAction<number>>;
  refreshList: () => void;
};

export function useCampaignsPageMutations({
  customerId,
  appliedCustomerId,
  createSectionOpen,
  setCreateSectionOpen,
  templates,
  draftTemplateId,
  setDraftTemplateId,
  draftCreateCustomerId,
  setDraftCreateCustomerId,
  draftCreateName,
  setDraftCreateName,
  draftBudgetLimitMicro,
  setDraftBudgetLimitMicro,
  setCreating,
  setActionError,
  setTemplatesRefreshToken,
  refreshList,
}: UseCampaignsPageMutationsArgs) {
  useEffect(() => {
    if (!createSectionOpen) {
      return;
    }
    setDraftCreateCustomerId(appliedCustomerId || customerId || '');
  }, [appliedCustomerId, createSectionOpen, customerId, setDraftCreateCustomerId]);

  useEffect(() => {
    if (templates.length === 0) {
      setDraftTemplateId('');
      return;
    }
    if (!templates.some((template) => template.id === draftTemplateId)) {
      setDraftTemplateId(templates[0]?.id ?? '');
    }
  }, [draftTemplateId, setDraftTemplateId, templates]);

  const onLoadTemplates = useCallback(() => {
    setTemplatesRefreshToken((value) => value + 1);
  }, [setTemplatesRefreshToken]);

  const onCreateCampaign = useCallback(async () => {
    const effectiveCustomerId = draftCreateCustomerId.trim() || customerId;
    if (!effectiveCustomerId || !draftTemplateId) {
      return;
    }

    const budgetRaw = draftBudgetLimitMicro.trim();
    let budgetLimitMicro: number | undefined;
    if (budgetRaw) {
      const parsed = Number.parseInt(budgetRaw, 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        setActionError(new Error('Budget must be a positive integer (micro units)'));
        return;
      }
      budgetLimitMicro = parsed;
    }

    setCreating(true);
    setActionError(undefined);
    try {
      await createSelfServeCampaign({
        customer_id: effectiveCustomerId,
        template_id: draftTemplateId,
        name: draftCreateName.trim() || undefined,
        budget_limit_micro: budgetLimitMicro,
      });
      setDraftCreateName('');
      setDraftBudgetLimitMicro('');
      setCreateSectionOpen(false);
      toast.success('Campaign created');
      refreshList();
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [
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
  ]);

  return {
    onLoadTemplates,
    onCreateCampaign,
  };
}
