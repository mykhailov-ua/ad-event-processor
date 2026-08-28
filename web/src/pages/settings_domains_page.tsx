import { useCallback, useEffect, useState } from 'react';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  addDomain,
  deleteDomain,
  fetchDomains,
  parkDomain,
  probeDomain,
  setupDomainSsl,
  type DomainHealth,
} from '../helpers/domains_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { DomainsPanel } from '../ui/settings/domains_panel.js';

export function SettingsDomainsPage() {
  const [domains, setDomains] = useState<DomainHealth[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busyHostname, setBusyHostname] = useState<string | null>(null);
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
      const [result, err] = await to(fetchDomains(ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setDomains(result ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  const runHostAction = useCallback(
    async (hostname: string, action: () => Promise<void>, successTitle: string) => {
      setBusyHostname(hostname);
      try {
        await action();
        pushToastMessage({ title: successTitle, message: hostname });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Action failed',
          message: err instanceof Error ? err.message : 'Request failed',
        });
      } finally {
        setBusyHostname(null);
      }
    },
    [reload]
  );

  return (
    <DomainsPanel
      domains={domains}
      loading={loading}
      error={error}
      busyHostname={busyHostname}
      onAdd={(hostname) => {
        void runHostAction(hostname, async () => {
          await addDomain(hostname);
        }, 'Domain added');
      }}
      onDelete={(hostname) => {
        void runHostAction(hostname, async () => {
          await deleteDomain(hostname);
        }, 'Domain deleted');
      }}
      onProbe={(hostname) => {
        void runHostAction(hostname, async () => {
          await probeDomain(hostname);
        }, 'Probe complete');
      }}
      onSslSetup={(hostname) => {
        void runHostAction(hostname, async () => {
          await setupDomainSsl(hostname);
        }, 'SSL setup started');
      }}
      onPark={(body) => {
        setBusyHostname(body.domain);
        void (async () => {
          try {
            await parkDomain(body);
            pushToastMessage({ title: 'Domain parked', message: body.domain });
            reload();
          } catch (err) {
            if (err instanceof ConfirmCancelledError) return;
            pushToastMessage({
              title: 'Park failed',
              message: err instanceof Error ? err.message : 'Park failed',
            });
          } finally {
            setBusyHostname(null);
          }
        })();
      }}
    />
  );
}
