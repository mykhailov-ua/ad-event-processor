import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  fetchFraudLabels,
  isValidFraudIPHash,
  postFraudLabel,
  postFraudLabelsBulk,
  type FraudManualLabelRow,
  type MLManualLabelDTO,
} from '../helpers/fraud_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { LabelsPanel } from '../ui/fraud/labels_panel.js';

const DEFAULT_LIMIT = 50;

function parseLimit(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_LIMIT;
  return Math.min(value, 100);
}

export function FraudLabelsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';
  const limit = parseLimit(searchParams.get('limit'));

  const [labels, setLabels] = useState<MLManualLabelDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [formBusy, setFormBusy] = useState(false);

  const [ipHash, setIpHash] = useState('');
  const [label, setLabel] = useState('1');
  const [reason, setReason] = useState('manual');
  const [bulkJson, setBulkJson] = useState('');

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    if (!customerId) {
      setLabels([]);
      setLoading(false);
      setError(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchFraudLabels({ customerId, limit }, ctrl.signal));
      if (cancelled) return;
      if (err && err.name === 'AbortError') return;
      if (err) {
        setError(err);
        setLabels([]);
      } else {
        setLabels(result ?? []);
        setError(null);
      }
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId, limit, reloadToken]);

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

  const onCreateLabel = useCallback(() => {
    if (!customerId) return;
    const normalized = ipHash.trim().toLowerCase();
    if (!isValidFraudIPHash(normalized)) {
      pushToastMessage({
        title: 'Invalid ip_hash',
        message: 'ip_hash must be 32 hexadecimal characters.',
      });
      return;
    }
    const labelNum = Number.parseInt(label, 10);
    if (labelNum !== 0 && labelNum !== 1) {
      pushToastMessage({ title: 'Invalid label', message: 'Label must be 0 or 1.' });
      return;
    }
    setFormBusy(true);
    void (async () => {
      try {
        await postFraudLabel(customerId, {
          ip_hash: normalized,
          label: labelNum,
          reason: reason.trim(),
        });
        pushToastMessage({ title: 'Label saved', message: normalized });
        setIpHash('');
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Save failed',
          message: err instanceof Error ? err.message : 'Save failed',
        });
      } finally {
        setFormBusy(false);
      }
    })();
  }, [customerId, ipHash, label, reason, reload]);

  const onBulkImport = useCallback(() => {
    if (!customerId) return;
    let parsed: { rows?: FraudManualLabelRow[] };
    try {
      parsed = JSON.parse(bulkJson) as { rows?: FraudManualLabelRow[] };
    } catch {
      pushToastMessage({ title: 'Invalid JSON', message: 'Bulk import must be valid JSON.' });
      return;
    }
    const rows = parsed.rows;
    if (!Array.isArray(rows) || rows.length === 0) {
      pushToastMessage({ title: 'Invalid bulk payload', message: 'rows array is required.' });
      return;
    }
    setFormBusy(true);
    void (async () => {
      try {
        const result = await postFraudLabelsBulk(customerId, rows);
        pushToastMessage({
          title: 'Bulk import complete',
          message: `${result.upserted} rows upserted`,
        });
        setBulkJson('');
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Bulk import failed',
          message: err instanceof Error ? err.message : 'Import failed',
        });
      } finally {
        setFormBusy(false);
      }
    })();
  }, [customerId, bulkJson, reload]);

  return (
    <LabelsPanel
      customerId={customerId}
      limit={limit}
      labels={labels}
      loading={loading}
      error={error}
      formBusy={formBusy}
      ipHash={ipHash}
      label={label}
      reason={reason}
      bulkJson={bulkJson}
      onCustomerApply={onCustomerApply}
      onLimitChange={(nextLimit) => patchParams({ limit: String(nextLimit) })}
      onIpHashChange={setIpHash}
      onLabelChange={setLabel}
      onReasonChange={setReason}
      onBulkJsonChange={setBulkJson}
      onCreateLabel={onCreateLabel}
      onBulkImport={onBulkImport}
    />
  );
}
