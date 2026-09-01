import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { createFraudOverride } from '@/api/fraud_api';
import { FraudOverrides } from '@/domains/fraud/fraud_overrides';
import { useSession } from '@/hooks/use_session';

export function FraudOverridesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);

  const customerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const [draftCustomerId, setDraftCustomerId] = useState(customerId);
  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [draftIpHash, setDraftIpHash] = useState('');
  const [draftIp, setDraftIp] = useState('');

  useEffect(() => {
    setDraftCustomerId(customerId);
  }, [customerId]);

  const onApplyCustomer = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftCustomerId.trim();
    if (trimmed) {
      next.set('customer_id', trimmed);
    } else {
      next.delete('customer_id');
    }
    setSearchParams(next, { replace: true });
  }, [draftCustomerId, searchParams, setSearchParams]);

  const onSubmit = useCallback(async () => {
    if (!customerId) {
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    const ipHash = draftIpHash.trim();
    const ip = draftIp.trim();
    if (!ipHash && !ip) {
      return;
    }
    try {
      await createFraudOverride(customerId, {
        campaign_id: draftCampaignId.trim() || undefined,
        ...(ipHash ? { ip_hash: ipHash } : {}),
        ...(ip ? { ip } : {}),
      });
      setSaveSuccess(true);
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [customerId, draftCampaignId, draftIpHash, draftIp]);

  return (
    <FraudOverrides
      customerId={customerId}
      draftCustomerId={draftCustomerId}
      draftCampaignId={draftCampaignId}
      draftIpHash={draftIpHash}
      draftIp={draftIp}
      saving={saving}
      saveError={saveError}
      saveSuccess={saveSuccess}
      onDraftCustomerIdChange={setDraftCustomerId}
      onDraftCampaignIdChange={setDraftCampaignId}
      onDraftIpHashChange={setDraftIpHash}
      onDraftIpChange={setDraftIp}
      onApplyCustomer={onApplyCustomer}
      onSubmit={onSubmit}
    />
  );
}
