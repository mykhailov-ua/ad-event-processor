import { useCallback, useEffect, useState } from 'react';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  applyPlatformSettings,
  fetchPlatformSettings,
  patchPlatformSettings,
  type PlatformSettingsPatch,
  type PlatformSettingsView,
} from '../helpers/settings_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { PlatformPanel } from '../ui/settings/platform_panel.js';

export function SettingsPage() {
  const [data, setData] = useState<PlatformSettingsView | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [saving, setSaving] = useState(false);
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
      const [result, err] = await to(fetchPlatformSettings(ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setData(result ?? null);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  const onSave = useCallback(
    async (patch: PlatformSettingsPatch) => {
      setSaving(true);
      try {
        const updated = await patchPlatformSettings(patch);
        setData(updated);
        pushToastMessage({ title: 'Platform settings saved', message: 'Draft updated in database.' });
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Save failed',
          message: err instanceof Error ? err.message : 'Save failed',
        });
      } finally {
        setSaving(false);
      }
    },
    []
  );

  const onApply = useCallback(async () => {
    setApplying(true);
    try {
      const result = await applyPlatformSettings();
      pushToastMessage({
        title: 'Applied to disk',
        message: result.written_path ?? 'ok',
      });
      reload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({
        title: 'Apply failed',
        message: err instanceof Error ? err.message : 'Apply failed',
      });
    } finally {
      setApplying(false);
    }
  }, [reload]);

  return (
    <PlatformPanel
      data={data}
      loading={loading}
      error={error}
      saving={saving}
      applying={applying}
      onSave={(patch) => {
        void onSave(patch);
      }}
      onApply={() => {
        void onApply();
      }}
    />
  );
}
