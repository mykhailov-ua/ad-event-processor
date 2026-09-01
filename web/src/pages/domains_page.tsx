import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import {
  addDomain,
  deleteDomain,
  listDomains,
  parkDomain,
  probeDomain,
  setupDomainSsl,
} from '@/api/domains_api';
import type { DomainSSLSetupResult } from '@/api/types';
import { DomainsDirectory } from '@/domains/creative/domains_directory';
import { useResource } from '@/hooks/use_resource';

export function DomainsPage() {
  const [reloadToken, setReloadToken] = useState(0);
  const { data, error, fetching } = useResource(
    (signal) => listDomains(signal),
    [reloadToken],
  );

  const [draftHostname, setDraftHostname] = useState('');
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [actionMessage, setActionMessage] = useState<string | undefined>(undefined);
  const [sslResult, setSslResult] = useState<DomainSSLSetupResult | undefined>(undefined);
  const [draftParkDomain, setDraftParkDomain] = useState('');
  const [draftParkZoneId, setDraftParkZoneId] = useState('');
  const [parkMessage, setParkMessage] = useState<string | undefined>(undefined);

  const bumpReload = useCallback(() => {
    setReloadToken((value) => value + 1);
  }, []);

  const onAddDomain = useCallback(() => {
    const hostname = draftHostname.trim();
    if (!hostname) {
      setActionError(new Error('Hostname is required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void addDomain({ hostname })
      .then(() => {
        setDraftHostname('');
        setActionMessage('Domain registered');
        toast.success('Domain registered');
        bumpReload();
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [bumpReload, draftHostname]);

  const onDeleteDomain = useCallback(
    (hostname: string) => {
      setActing(true);
      setActionError(undefined);
      setActionMessage(undefined);
      void deleteDomain(hostname)
        .then(() => {
          setActionMessage(`Deleted ${hostname}`);
          toast.success(`Deleted ${hostname}`);
          bumpReload();
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpReload],
  );

  const onProbeDomain = useCallback(
    (hostname: string) => {
      setActing(true);
      setActionError(undefined);
      setActionMessage(undefined);
      void probeDomain(hostname)
        .then(() => {
          setActionMessage(`Probe completed for ${hostname}`);
          toast.success(`Probe completed for ${hostname}`);
          bumpReload();
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpReload],
  );

  const onSetupSsl = useCallback((hostname: string) => {
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void setupDomainSsl(hostname)
      .then((result) => {
        setSslResult(result);
        setActionMessage(`SSL setup for ${hostname}`);
        toast.success(`SSL setup for ${hostname}`);
        bumpReload();
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [bumpReload]);

  const onParkDomain = useCallback(() => {
    const domain = draftParkDomain.trim();
    const cloudflareZoneId = draftParkZoneId.trim();
    if (!domain || !cloudflareZoneId) {
      setActionError(new Error('Domain and Cloudflare zone ID are required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setParkMessage(undefined);
    void parkDomain({ domain, cloudflare_zone_id: cloudflareZoneId })
      .then((result) => {
        setParkMessage(
          result.hostname
            ? `Parked ${result.hostname} (${result.ssl_status ?? 'ssl pending'})`
            : 'Domain parked',
        );
        toast.success(result.hostname ? `Parked ${result.hostname}` : 'Domain parked');
        bumpReload();
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [bumpReload, draftParkDomain, draftParkZoneId]);

  return (
    <DomainsDirectory
      items={data ?? []}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftHostname={draftHostname}
      acting={acting}
      actionError={actionError}
      actionMessage={actionMessage}
      sslResult={sslResult}
      onDraftHostnameChange={setDraftHostname}
      onAddDomain={onAddDomain}
      onDeleteDomain={onDeleteDomain}
      onProbeDomain={onProbeDomain}
      onSetupSsl={onSetupSsl}
      draftParkDomain={draftParkDomain}
      draftParkZoneId={draftParkZoneId}
      onDraftParkDomainChange={setDraftParkDomain}
      onDraftParkZoneIdChange={setDraftParkZoneId}
      onParkDomain={onParkDomain}
      parkMessage={parkMessage}
    />
  );
}
