import { useEffect, useState } from 'react';
import * as auth from '../helpers/auth.js';
import { fetchSelfServeTemplates, type SelfServeTemplate } from '../helpers/selfserve_api.js';
import { to } from '../lib/to.js';
import { CampaignCreatePanel } from '../ui/selfserve/campaign_create_panel.js';
import { SelfServeShell } from '../ui/selfserve/selfserve_shell.js';

export function SelfServeCampaignCreatePage() {
  const customerId = auth.getUser()?.customer_id ?? '';
  const [templates, setTemplates] = useState<SelfServeTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    const ctrl = new AbortController();
    setLoading(true);
    setError(null);
    void (async () => {
      const [data, err] = await to(fetchSelfServeTemplates(customerId || undefined, ctrl.signal));
      if (ctrl.signal.aborted) return;
      if (err) setError(err);
      else setTemplates(data?.items ?? []);
      setLoading(false);
    })();
    return () => ctrl.abort();
  }, [customerId]);

  return (
    <SelfServeShell>
      <CampaignCreatePanel
        templates={templates}
        loading={loading}
        error={error}
        customerId={customerId || undefined}
      />
    </SelfServeShell>
  );
}
