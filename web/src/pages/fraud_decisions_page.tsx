import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  fetchFraudDecision,
  isValidFraudIPHash,
  type FraudDecisionDTO,
} from '../helpers/fraud_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { DecisionsPanel } from '../ui/fraud/decisions_panel.js';

const DEFAULT_HOURS = 24;

function parseHours(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_HOURS;
  return Math.min(value, 168);
}

export function FraudDecisionsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';
  const ipHashParam = (searchParams.get('ip_hash') ?? '').trim().toLowerCase();
  const hoursParam = searchParams.get('hours') ?? String(DEFAULT_HOURS);
  const campaignIdParam = searchParams.get('campaign_id') ?? '';

  const [ipHashDraft, setIpHashDraft] = useState(ipHashParam);
  const [hoursDraft, setHoursDraft] = useState(hoursParam);
  const [campaignIdDraft, setCampaignIdDraft] = useState(campaignIdParam);

  const [decision, setDecision] = useState<FraudDecisionDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);

  const lookupReady = useMemo(
    () => Boolean(customerId && ipHashParam && isValidFraudIPHash(ipHashParam)),
    [customerId, ipHashParam]
  );

  useEffect(() => {
    setIpHashDraft(ipHashParam);
    setHoursDraft(hoursParam);
    setCampaignIdDraft(campaignIdParam);
  }, [ipHashParam, hoursParam, campaignIdParam]);

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

  const onLookup = useCallback(() => {
    const normalized = ipHashDraft.trim().toLowerCase();
    if (!isValidFraudIPHash(normalized)) {
      pushToastMessage({
        title: 'Invalid ip_hash',
        message: 'ip_hash must be 32 hexadecimal characters.',
      });
      return;
    }
    patchParams({
      ip_hash: normalized,
      hours: hoursDraft.trim() || String(DEFAULT_HOURS),
      campaign_id: campaignIdDraft.trim() || null,
    });
    setReloadToken((token) => token + 1);
  }, [ipHashDraft, hoursDraft, campaignIdDraft, patchParams]);

  useEffect(() => {
    if (!lookupReady) {
      setDecision(null);
      setError(null);
      setLoading(false);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(
        fetchFraudDecision(
          {
            customerId,
            ipHash: ipHashParam,
            hours: parseHours(hoursParam),
            campaignId: campaignIdParam || undefined,
          },
          ctrl.signal
        )
      );
      if (cancelled) return;
      if (err && err.name === 'AbortError') return;
      if (err) {
        setError(err);
        setDecision(null);
      } else {
        setDecision(result ?? null);
        setError(null);
      }
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [lookupReady, customerId, ipHashParam, hoursParam, campaignIdParam, reloadToken]);

  return (
    <DecisionsPanel
      customerId={customerId}
      ipHash={ipHashDraft}
      hours={hoursDraft}
      campaignId={campaignIdDraft}
      decision={decision}
      loading={loading}
      error={error}
      lookupReady={lookupReady}
      onCustomerApply={onCustomerApply}
      onLookupDraftChange={(patch) => {
        if (patch.ipHash != null) setIpHashDraft(patch.ipHash);
        if (patch.hours != null) setHoursDraft(patch.hours);
        if (patch.campaignId != null) setCampaignIdDraft(patch.campaignId);
      }}
      onLookup={onLookup}
    />
  );
}
