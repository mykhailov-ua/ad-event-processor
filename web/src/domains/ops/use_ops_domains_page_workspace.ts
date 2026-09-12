// TLS/domain ops: separate lazy GET lanes for rotation list and per-host TLS ask.
import { useState } from 'react';

import { ApiError } from '@/api/client';
import { checkOpsTlsAllowed, getOpsDomainRotation } from '@/api/ops_api';
import type { OpsDomainRotationResponse } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';
import { requireNonEmpty } from '@/lib/admin_validation_error';

function readSslSetupError(rotation: OpsDomainRotationResponse | undefined): Error | undefined {
  if (rotation == null || typeof rotation !== 'object') {
    return undefined;
  }
  const sslSetup = (rotation as { ssl_setup?: { available?: boolean; message?: string } })
    .ssl_setup;
  if (!sslSetup || sslSetup.available !== false) {
    return undefined;
  }
  const message = sslSetup.message ?? 'ssl setup script not found';
  return new ApiError(501, 'NOT_IMPLEMENTED', message);
}

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsDomainsPageWorkspace() {
  const [draftHostname, setDraftHostname] = useState('');
  const [rotationLoadToken, setRotationLoadToken] = useState(0);
  const [tlsHostLoadToken, setTlsHostLoadToken] = useState(0);
  const [lookupHostname, setLookupHostname] = useState('');
  const [tlsHostValidationError, setTlsHostValidationError] = useState<Error | undefined>();

  const sslSetupProbe = useResource((signal) => getOpsDomainRotation(signal), []);

  const rotationResource = useResource(
    (signal) => {
      if (rotationLoadToken === 0) {
        return skipLazyFetch();
      }
      return getOpsDomainRotation(signal);
    },
    [rotationLoadToken]
  );

  const tlsHostResource = useResource(
    (signal) => {
      if (tlsHostLoadToken === 0 || !lookupHostname) {
        return skipLazyFetch();
      }
      return checkOpsTlsAllowed(lookupHostname, signal);
    },
    [tlsHostLoadToken, lookupHostname]
  );

  const onLoadRotation = useCoalescedBumpRefresh(() => {
    setRotationLoadToken((value) => value + 1);
  }, rotationResource.fetching);

  const onLookupTlsHost = useCoalescedCallback(
    () => {
      const hostnameResult = requireNonEmpty(draftHostname, 'Hostname', 'hostname');
      if (!hostnameResult.ok) {
        setTlsHostValidationError(hostnameResult.error);
        return;
      }
      setTlsHostValidationError(undefined);
      setLookupHostname(hostnameResult.value);
      setTlsHostLoadToken((value) => value + 1);
    },
    {
      inFlightGuard: true,
      inFlight: tlsHostResource.fetching,
    }
  );

  return {
    rotation: rotationResource.data,
    tlsHost: tlsHostResource.data,
    draftHostname,
    fetchingRotation: rotationResource.fetching,
    fetchingTlsHost: tlsHostResource.fetching,
    rotationError: rotationResource.error,
    sslSetupError: readSslSetupError(sslSetupProbe.data ?? rotationResource.data),
    tlsHostError: tlsHostValidationError ?? tlsHostResource.error,
    hasRotationSnapshot: rotationResource.data != null,
    onDraftHostnameChange: setDraftHostname,
    onLoadRotation,
    onLookupTlsHost,
  };
}
