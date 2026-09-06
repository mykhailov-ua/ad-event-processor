import { useCallback, useState } from 'react';

import { acceptEula, getEulaStatus } from '@/api/platform_api';
import { useResource } from '@/api/use_resource';
import { fetchMetaCached } from '@/lib/get_meta_cache';
import { useSession } from '@/hooks/use_session';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export type EulaGateSnapshot = {
  required: boolean;
  accepted: boolean;
  version: string | undefined;
  eulaText: string | undefined;
};

async function loadEulaGateSnapshot(
  bootstrapRequired: boolean | undefined,
  bootstrapAccepted: boolean | undefined,
  bootstrapVersion: string | undefined,
  signal: AbortSignal
): Promise<EulaGateSnapshot> {
  if (bootstrapRequired !== undefined) {
    const required = bootstrapRequired === true;
    const accepted = bootstrapAccepted === true;
    let version = bootstrapVersion;
    let eulaText: string | undefined;

    if (required && !accepted) {
      const fullStatus = await getEulaStatus(signal);
      version = fullStatus.version ?? bootstrapVersion;
      eulaText = fullStatus.text;
    }

    return { required, accepted, version, eulaText };
  }

  const meta = await fetchMetaCached(signal);
  const required = meta.eula_required === true;
  const accepted = meta.eula_accepted === true;
  let version = meta.eula_version;
  let eulaText: string | undefined;

  if (required && !accepted) {
    const fullStatus = await getEulaStatus(signal);
    version = fullStatus.version ?? meta.eula_version;
    eulaText = fullStatus.text;
  }

  return { required, accepted, version, eulaText };
}

export function useEulaGate() {
  const {
    user,
    loading: sessionLoading,
    eulaRequired: bootstrapRequired,
    eulaAccepted: bootstrapAccepted,
    eulaVersion: bootstrapVersion,
  } = useSession();

  const [accepting, setAccepting] = useState(false);
  const [acceptError, setAcceptError] = useState<Error | undefined>();
  const [acceptedOverride, setAcceptedOverride] = useState<boolean | undefined>();
  const [requiredOverride, setRequiredOverride] = useState<boolean | undefined>();
  const [versionOverride, setVersionOverride] = useState<string | undefined>();
  const [eulaTextOverride, setEulaTextOverride] = useState<string | undefined>();

  const statusResource = useResource(
    (signal) => {
      if (sessionLoading) {
        return skipLazyFetch();
      }
      return loadEulaGateSnapshot(bootstrapRequired, bootstrapAccepted, bootstrapVersion, signal);
    },
    [bootstrapAccepted, bootstrapRequired, bootstrapVersion, sessionLoading]
  );

  const required = requiredOverride ?? statusResource.data?.required ?? false;
  const accepted = acceptedOverride ?? statusResource.data?.accepted ?? false;
  const version = versionOverride ?? statusResource.data?.version;
  const eulaText = eulaTextOverride ?? statusResource.data?.eulaText;
  const loading = sessionLoading || (statusResource.fetching && statusResource.data == null);
  const blocked = !loading && !statusResource.error && required && !accepted;
  const canAccept = user?.permissions?.includes('settings:write') ?? false;

  const onAccept = useCallback(async () => {
    const acceptVersion = version?.trim();
    if (!acceptVersion) {
      setAcceptError(new Error('EULA version missing from server status'));
      return;
    }
    setAccepting(true);
    setAcceptError(undefined);
    try {
      const next = await acceptEula({ version: acceptVersion });
      setAcceptedOverride(next.accepted === true);
      setRequiredOverride(next.required === true);
      setVersionOverride(next.version);
      if (next.text) {
        setEulaTextOverride(next.text);
      }
    } catch (err: unknown) {
      setAcceptError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setAccepting(false);
    }
  }, [version]);

  return {
    loading,
    error: statusResource.error,
    required,
    accepted,
    version,
    eulaText,
    accepting,
    acceptError,
    blocked,
    canAccept,
    onAccept: () => {
      void onAccept();
    },
  };
}
