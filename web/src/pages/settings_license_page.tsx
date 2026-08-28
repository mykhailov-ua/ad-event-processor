import { useCallback, useEffect, useState } from 'react';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  applyLicenseToken,
  fetchLicenseStatus,
  type LicenseStatus,
} from '../helpers/settings_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { LicensePanel } from '../ui/settings/license_panel.js';

export function SettingsLicensePage() {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [applying, setApplying] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchLicenseStatus(ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setStatus(result ?? null);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  const onApply = useCallback(
    async (token: string) => {
      setApplying(true);
      try {
        const updated = await applyLicenseToken(token);
        setStatus(updated);
        pushToastMessage({ title: 'License applied', message: updated.state ?? 'ok' });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'License apply failed',
          message: err instanceof Error ? err.message : 'Apply failed',
        });
      } finally {
        setApplying(false);
      }
    },
    [reload]
  );

  return (
    <LicensePanel
      status={status}
      loading={loading}
      error={error}
      applying={applying}
      onApply={(token) => {
        void onApply(token);
      }}
    />
  );
}
