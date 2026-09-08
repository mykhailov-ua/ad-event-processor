// tracking domains: list + add/delete/probe/park/SSL/bulk/burn actions; per-action acting guard.
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  addDomain,
  burnDomain,
  deleteDomain,
  getDomainBulkJob,
  listCloudflareZones,
  listDomains,
  parkDomain,
  probeDomain,
  setupDomainSsl,
  setupWildcardSSL,
  startBulkDomainPark,
  startBulkDomainSSL,
} from '@/api/domains_api';
import type {
  CloudflareZone,
  DomainBulkJobStatus,
  DomainHealth,
  DomainSSLSetupResult,
  WildcardSSLResponse,
} from '@/api/types';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export type DomainHealthFilter = 'all' | 'healthy' | 'degraded' | 'burned';

function matchesDomainFilter(row: DomainHealth, filter: DomainHealthFilter): boolean {
  if (filter === 'all') {
    return true;
  }
  if (filter === 'burned') {
    return row.pool_status === 'banned';
  }
  if (filter === 'healthy') {
    return row.health_status === 'healthy' && row.pool_status !== 'banned';
  }
  if (filter === 'degraded') {
    return (
      (row.health_status === 'degraded' || row.health_status === 'down') &&
      row.pool_status !== 'banned'
    );
  }
  return true;
}

export function useDomainsPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource((signal) => listDomains(signal), [refreshToken]);

  const [draftHostname, setDraftHostname] = useState('');
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [actionMessage, setActionMessage] = useState<string | undefined>(undefined);
  const [sslResult, setSslResult] = useState<DomainSSLSetupResult | undefined>(undefined);
  const [draftParkDomain, setDraftParkDomain] = useState('');
  const [draftParkZoneId, setDraftParkZoneId] = useState('');
  const [parkMessage, setParkMessage] = useState<string | undefined>(undefined);
  const [wildcardOpen, setWildcardOpen] = useState(false);
  const [draftWildcardZoneId, setDraftWildcardZoneId] = useState('');
  const [draftWildcardZoneName, setDraftWildcardZoneName] = useState('');
  const [draftWildcardIncludeApex, setDraftWildcardIncludeApex] = useState(false);
  const [cloudflareZones, setCloudflareZones] = useState<CloudflareZone[]>([]);
  const [zonesError, setZonesError] = useState<Error | undefined>(undefined);
  const [wildcardResult, setWildcardResult] = useState<WildcardSSLResponse | undefined>(undefined);
  const [healthFilter, setHealthFilter] = useState<DomainHealthFilter>('all');
  const [bulkOpen, setBulkOpen] = useState(false);
  const [draftBulkText, setDraftBulkText] = useState('');
  const [draftBulkCsv, setDraftBulkCsv] = useState('');
  const [draftBulkZoneId, setDraftBulkZoneId] = useState('');
  const [bulkJobId, setBulkJobId] = useState('');
  const [bulkJob, setBulkJob] = useState<DomainBulkJobStatus | undefined>(undefined);
  const [bulkJobError, setBulkJobError] = useState<Error | undefined>(undefined);
  const [pollToken, setPollToken] = useState(0);
  const [burnOpen, setBurnOpen] = useState(false);
  const [burnHostname, setBurnHostname] = useState('');
  const [burnDeleteCloudflare, setBurnDeleteCloudflare] = useState(false);

  const listBusy = fetching || acting;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const bumpReload = bumpRefreshCoalesced;

  const filteredItems = useMemo(() => {
    const rows = data ?? [];
    return rows.filter((row) => matchesDomainFilter(row, healthFilter));
  }, [data, healthFilter]);

  useEffect(() => {
    if (!bulkJobId) {
      return;
    }
    if (bulkJob?.status === 'completed' || bulkJob?.status === 'failed') {
      return;
    }
    const timer = window.setInterval(() => {
      setPollToken((value) => value + 1);
    }, 1500);
    return () => window.clearInterval(timer);
  }, [bulkJob?.status, bulkJobId]);

  useEffect(() => {
    if (!bulkJobId) {
      return;
    }
    let cancelled = false;
    void getDomainBulkJob(bulkJobId)
      .then((status) => {
        if (cancelled) {
          return;
        }
        setBulkJob(status);
        setBulkJobError(undefined);
        if (status.status === 'completed' || status.status === 'failed') {
          bumpReload();
          if (status.status === 'completed') {
            toast.success(
              `Bulk job finished (${status.completed - status.failed}/${status.total} ok)`
            );
          } else {
            toast.error(status.error ?? 'Bulk job failed');
          }
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setBulkJobError(err instanceof Error ? err : new Error(String(err)));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [bulkJobId, pollToken, bumpReload]);

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
      if (acting) {
        return;
      }
      if (!confirmDestructiveAction(`Delete domain "${hostname}"?`)) {
        return;
      }
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
          const nextError = mutationError(err);
          setActionError(nextError);
          toast.error(nextError.message);
        })
        .finally(() => {
          setActing(false);
        });
    },
    [acting, bumpReload]
  );

  const onOpenBurnDialog = useCallback((hostname: string) => {
    setBurnHostname(hostname);
    setBurnDeleteCloudflare(false);
    setBurnOpen(true);
  }, []);

  const onConfirmBurn = useCallback(() => {
    const hostname = burnHostname.trim();
    if (!hostname || acting) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void burnDomain(hostname, { delete_cloudflare: burnDeleteCloudflare })
      .then(() => {
        setActionMessage(`Burned ${hostname}`);
        toast.success(`Burned ${hostname}`);
        setBurnOpen(false);
        bumpReload();
      })
      .catch((err: unknown) => {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      })
      .finally(() => {
        setActing(false);
      });
  }, [acting, burnDeleteCloudflare, burnHostname, bumpReload]);

  const onBurnDomain = onOpenBurnDialog;

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
    [bumpReload]
  );

  const onSetupSsl = useCallback(
    (hostname: string) => {
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
    },
    [bumpReload]
  );

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
            : 'Domain parked'
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

  const parseBulkHostnames = useCallback(() => {
    return draftBulkText
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean);
  }, [draftBulkText]);

  const hasBulkInput = useCallback(() => {
    return parseBulkHostnames().length > 0 || draftBulkCsv.trim().length > 0;
  }, [draftBulkCsv, parseBulkHostnames]);

  const onBulkCsvFileSelected = useCallback((file: File | undefined) => {
    if (!file) {
      return;
    }
    void file
      .text()
      .then((text) => {
        setDraftBulkCsv(text);
        setActionError(undefined);
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      });
  }, []);

  const onOpenBulkDialog = useCallback(() => {
    setBulkOpen(true);
    setDraftBulkCsv('');
    setBulkJob(undefined);
    setBulkJobError(undefined);
    setBulkJobId('');
    setZonesError(undefined);
    void listCloudflareZones()
      .then((zones) => setCloudflareZones(zones))
      .catch((err: unknown) => {
        setZonesError(err instanceof Error ? err : new Error(String(err)));
      });
  }, []);

  const onStartBulkPark = useCallback(() => {
    const cloudflareZoneId = draftBulkZoneId.trim();
    const hostnames = parseBulkHostnames();
    const csv = draftBulkCsv.trim();
    if (!cloudflareZoneId || !hasBulkInput()) {
      setActionError(new Error('Zone ID and hostnames (paste or CSV file) are required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    void startBulkDomainPark({
      hostnames,
      csv: csv || undefined,
      cloudflare_zone_id: cloudflareZoneId,
    })
      .then((job) => {
        setBulkJobId(job.job_id);
        setBulkJob(job);
        toast.success('Bulk park job started');
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [draftBulkCsv, draftBulkZoneId, hasBulkInput, parseBulkHostnames]);

  const onStartBulkSSL = useCallback(() => {
    const hostnames = parseBulkHostnames();
    const csv = draftBulkCsv.trim();
    if (!hasBulkInput()) {
      setActionError(new Error('At least one hostname (paste or CSV file) is required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    void startBulkDomainSSL({ hostnames, csv: csv || undefined })
      .then((job) => {
        setBulkJobId(job.job_id);
        setBulkJob(job);
        toast.success('Bulk SSL job started');
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [draftBulkCsv, hasBulkInput, parseBulkHostnames]);

  const onOpenWildcardWizard = useCallback(() => {
    setWildcardOpen(true);
    setZonesError(undefined);
    setWildcardResult(undefined);
    void listCloudflareZones()
      .then((zones) => setCloudflareZones(zones))
      .catch((err: unknown) => {
        setZonesError(err instanceof Error ? err : new Error(String(err)));
      });
  }, []);

  const onSetupWildcardSSL = useCallback(() => {
    const cloudflareZoneId = draftWildcardZoneId.trim();
    const zoneName = draftWildcardZoneName.trim();
    if (!cloudflareZoneId || !zoneName) {
      setActionError(new Error('Cloudflare zone and zone name are required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setWildcardResult(undefined);
    void setupWildcardSSL({
      cloudflare_zone_id: cloudflareZoneId,
      zone_name: zoneName,
      include_apex: draftWildcardIncludeApex,
    })
      .then((result) => {
        setWildcardResult(result);
        if (result.acme_state === 'valid') {
          toast.success(`Wildcard cert issued for ${result.wildcard_hostname}`);
          setWildcardOpen(false);
          bumpReload();
        } else {
          setActionError(new Error(result.message ?? 'Wildcard SSL issuance failed'));
        }
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [bumpReload, draftWildcardIncludeApex, draftWildcardZoneId, draftWildcardZoneName]);

  return {
    items: filteredItems,
    fetching,
    error,
    hasSnapshot: data != null,
    draftHostname,
    acting,
    actionError,
    actionMessage,
    sslResult,
    onDraftHostnameChange: setDraftHostname,
    onAddDomain,
    onDeleteDomain,
    onBurnDomain,
    onOpenBurnDialog,
    burnOpen,
    onBurnOpenChange: setBurnOpen,
    burnHostname,
    burnDeleteCloudflare,
    onBurnDeleteCloudflareChange: setBurnDeleteCloudflare,
    onConfirmBurn,
    onProbeDomain,
    onSetupSsl,
    draftParkDomain,
    draftParkZoneId,
    onDraftParkDomainChange: setDraftParkDomain,
    onDraftParkZoneIdChange: setDraftParkZoneId,
    onParkDomain,
    parkMessage,
    healthFilter,
    onHealthFilterChange: setHealthFilter,
    bulkOpen,
    onBulkOpenChange: setBulkOpen,
    onOpenBulkDialog,
    draftBulkText,
    onDraftBulkTextChange: setDraftBulkText,
    draftBulkCsvLoaded: draftBulkCsv.trim().length > 0,
    onBulkCsvFileSelected,
    draftBulkZoneId,
    onDraftBulkZoneIdChange: setDraftBulkZoneId,
    onStartBulkPark,
    onStartBulkSSL,
    bulkJob,
    bulkJobError,
    wildcardOpen,
    onWildcardOpenChange: setWildcardOpen,
    onOpenWildcardWizard,
    cloudflareZones,
    zonesError,
    draftWildcardZoneId,
    draftWildcardZoneName,
    draftWildcardIncludeApex,
    onDraftWildcardZoneIdChange: setDraftWildcardZoneId,
    onDraftWildcardZoneNameChange: setDraftWildcardZoneName,
    onDraftWildcardIncludeApexChange: setDraftWildcardIncludeApex,
    onSetupWildcardSSL,
    wildcardResult,
  };
}
