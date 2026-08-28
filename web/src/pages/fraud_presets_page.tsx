import { useEffect, useState } from 'react';
import { fetchFraudPresets, type FraudPolicyPresetDTO } from '../helpers/fraud_api.js';
import { to } from '../lib/to.js';
import { PresetsPanel } from '../ui/fraud/presets_panel.js';

export function FraudPresetsPage() {
  const [presets, setPresets] = useState<FraudPolicyPresetDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchFraudPresets(ctrl.signal));
      if (cancelled) return;
      if (err && err.name === 'AbortError') return;
      if (err) {
        setError(err);
        setPresets([]);
      } else {
        setPresets(result ?? []);
        setError(null);
      }
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, []);

  return <PresetsPanel presets={presets} loading={loading} error={error} />;
}
