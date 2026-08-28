import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  blockBlacklist,
  fetchBlacklist,
  unblockBlacklist,
  type BlacklistEntry,
} from '../helpers/ops_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { OpsBlacklistPanel } from '../ui/ops/ops_blacklist_panel.js';

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

export function OpsBlacklistPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));

  const [items, setItems] = useState<BlacklistEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);

  const [ip, setIp] = useState('');
  const [reason, setReason] = useState('manual');
  const [formBusy, setFormBusy] = useState(false);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchBlacklist({ limit, offset }, ctrl.signal));
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
  }, [limit, offset, reloadToken]);

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

  const onBlock = useCallback(() => {
    const trimmedIp = ip.trim();
    if (!trimmedIp) return;
    const source = reason.trim() || 'manual';
    setFormBusy(true);
    void (async () => {
      try {
        await blockBlacklist({ ip: trimmedIp, source });
        pushToastMessage({ title: 'IP blocked', message: trimmedIp });
        setIp('');
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Block failed',
          message: err instanceof Error ? err.message : 'Block failed',
        });
      } finally {
        setFormBusy(false);
      }
    })();
  }, [ip, reason, reload]);

  const onUnblock = useCallback(
    (entryIp: string, entrySource: string) => {
      setFormBusy(true);
      void (async () => {
        try {
          await unblockBlacklist({ ip: entryIp, source: entrySource });
          pushToastMessage({ title: 'IP unblocked', message: entryIp });
          reload();
        } catch (err) {
          if (err instanceof ConfirmCancelledError) return;
          pushToastMessage({
            title: 'Unblock failed',
            message: err instanceof Error ? err.message : 'Unblock failed',
          });
        } finally {
          setFormBusy(false);
        }
      })();
    },
    [reload]
  );

  return (
    <OpsBlacklistPanel
      items={items}
      total={total}
      limit={limit}
      offset={offset}
      loading={loading}
      error={error}
      ip={ip}
      reason={reason}
      formBusy={formBusy}
      onIpChange={setIp}
      onReasonChange={setReason}
      onBlock={onBlock}
      onUnblock={onUnblock}
      onOffsetChange={(nextOffset) => patchParams({ offset: String(nextOffset) })}
    />
  );
}
