import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { fetchFraudIntegrations, type FraudIntegrationDTO } from '../helpers/fraud_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { to } from '../lib/to.js';
import { IntegrationsPanel } from '../ui/fraud/integrations_panel.js';

export function FraudIntegrationsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';

  const [integrations, setIntegrations] = useState<FraudIntegrationDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  useEffect(() => {
    if (!customerId) {
      setIntegrations([]);
      setLoading(false);
      setError(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchFraudIntegrations(customerId, ctrl.signal));
      if (cancelled) return;
      if (err && err.name === 'AbortError') return;
      if (err) {
        setError(err);
        setIntegrations([]);
      } else {
        setIntegrations(result ?? []);
        setError(null);
      }
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId]);

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

  return (
    <IntegrationsPanel
      customerId={customerId}
      integrations={integrations}
      loading={loading}
      error={error}
      onCustomerApply={onCustomerApply}
    />
  );
}
