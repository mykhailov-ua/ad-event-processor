import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { isValidFraudIPHash, postFraudOverride } from '../helpers/fraud_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { OverridesPanel } from '../ui/fraud/overrides_panel.js';

export function FraudOverridesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';

  const [campaignId, setCampaignId] = useState('');
  const [ip, setIp] = useState('');
  const [ipHash, setIpHash] = useState('');
  const [error, setError] = useState<unknown>(null);
  const [formBusy, setFormBusy] = useState(false);

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  const patchParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null || value === '') next.delete(key);
        else next.set(key, value);
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      patchParams({ customer_id: nextCustomerId || null });
    },
    [patchParams]
  );

  const onSubmit = useCallback(() => {
    if (!customerId) return;
    const trimmedIp = ip.trim();
    const normalizedHash = ipHash.trim().toLowerCase();
    if (!trimmedIp && !normalizedHash) {
      pushToastMessage({ title: 'Identifier required', message: 'Provide IP or ip_hash.' });
      return;
    }
    if (normalizedHash && !isValidFraudIPHash(normalizedHash)) {
      pushToastMessage({
        title: 'Invalid ip_hash',
        message: 'ip_hash must be 32 hexadecimal characters.',
      });
      return;
    }
    setFormBusy(true);
    setError(null);
    void (async () => {
      try {
        const body: {
          campaign_id?: string;
          ip?: string;
          ip_hash?: string;
        } = {};
        if (campaignId.trim()) body.campaign_id = campaignId.trim();
        if (trimmedIp) body.ip = trimmedIp;
        if (normalizedHash) body.ip_hash = normalizedHash;
        await postFraudOverride(customerId, body);
        pushToastMessage({
          title: 'False positive override queued',
          message: trimmedIp || normalizedHash,
        });
        setIp('');
        setIpHash('');
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        setError(err);
        pushToastMessage({
          title: 'Override failed',
          message: err instanceof Error ? err.message : 'Override failed',
        });
      } finally {
        setFormBusy(false);
      }
    })();
  }, [customerId, campaignId, ip, ipHash]);

  return (
    <OverridesPanel
      customerId={customerId}
      campaignId={campaignId}
      ip={ip}
      ipHash={ipHash}
      error={error}
      formBusy={formBusy}
      onCustomerApply={onCustomerApply}
      onCampaignIdChange={setCampaignId}
      onIpChange={setIp}
      onIpHashChange={setIpHash}
      onSubmit={onSubmit}
    />
  );
}
