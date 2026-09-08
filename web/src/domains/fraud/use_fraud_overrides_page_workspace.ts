// L3 per-IP fraud override form: customer scope from useCustomerScope; POST on submit only.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createFraudOverride } from '@/api/fraud_api';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { mutationError } from '@/lib/mutation_audit';

export function useFraudOverridesPageWorkspace() {
  const {
    appliedCustomerId: customerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [draftIpHash, setDraftIpHash] = useState('');
  const [draftIp, setDraftIp] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);

  const onApplyCustomer = useCoalescedCallback(applyCustomerScope, {});

  const onSubmit = useCallback(async () => {
    if (saving) {
      return;
    }
    if (!customerId) {
      setSaveError(new Error('Apply a customer before creating an override.'));
      setSaveSuccess(false);
      return;
    }
    const ipHash = draftIpHash.trim();
    const ip = draftIp.trim();
    if (!ipHash && !ip) {
      setSaveError(new Error('IP hash or IP address is required.'));
      setSaveSuccess(false);
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await createFraudOverride(customerId, {
        campaign_id: draftCampaignId.trim() || undefined,
        ...(ipHash ? { ip_hash: ipHash } : {}),
        ...(ip ? { ip } : {}),
      });
      setSaveSuccess(true);
      toast.success('Fraud override created');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setSaveError(nextError);
      toast.error(nextError.message);
    } finally {
      setSaving(false);
    }
  }, [customerId, draftCampaignId, draftIpHash, draftIp, saving]);

  const onSubmitCoalesced = useCoalescedCallback(
    () => {
      void onSubmit();
    },
    {
      inFlightGuard: true,
      inFlight: saving,
    }
  );

  return {
    customerId,
    draftCustomerId,
    draftCampaignId,
    draftIpHash,
    draftIp,
    saving,
    saveError,
    saveSuccess,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftCampaignIdChange: setDraftCampaignId,
    onDraftIpHashChange: setDraftIpHash,
    onDraftIpChange: setDraftIp,
    onApplyCustomer,
    onSubmit: onSubmitCoalesced,
  };
}
