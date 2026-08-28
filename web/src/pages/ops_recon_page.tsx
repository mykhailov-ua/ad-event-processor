import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { buildReconRunsUrl, fetchReconRuns, type ReconRun } from '../helpers/ops_api.js';
import { to } from '../lib/to.js';
import { OpsReconPanel } from '../ui/ops/ops_recon_panel.js';

const DEFAULT_LIMIT = 50;

function parseLimit(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_LIMIT;
  return Math.min(value, 200);
}

function parseOffset(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value < 0) return 0;
  return value;
}

export function OpsReconPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));
  const serviceFilter = searchParams.get('service') ?? '';
  const [draftService, setDraftService] = useState(serviceFilter);

  const [items, setItems] = useState<ReconRun[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

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

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(
        fetchReconRuns(
          {
            limit,
            offset,
            service: serviceFilter || undefined,
          },
          ctrl.signal
        )
      );
      if (cancelled) return;
      if (err) {
        if (err.name === 'AbortError') return;
        setError(err);
        setLoading(false);
        return;
      }
      setItems(result?.items ?? []);
      setTotal(result?.total ?? 0);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [limit, offset, serviceFilter]);

  return (
    <OpsReconPanel
      items={items}
      total={total}
      limit={limit}
      offset={offset}
      serviceFilter={draftService}
      loading={loading}
      error={error}
      onServiceFilterChange={setDraftService}
      onApplyService={() => patchParams({ service: draftService.trim() || null, offset: '0' })}
      onOffsetChange={(nextOffset) => patchParams({ offset: String(nextOffset) })}
    />
  );
}
