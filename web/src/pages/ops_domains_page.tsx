import { useCallback, useState } from 'react';

import {
  getOpsDomainRotation,
  getOpsTlsAllowedHost,
  getOpsTlsAllowedList,
} from '@/api/ops_api';
import { OpsDomains } from '@/domains/ops/ops_domains';

export function OpsDomainsPage() {
  const [draftHostname, setDraftHostname] = useState('');
  const [rotation, setRotation] = useState<Record<string, unknown> | undefined>();
  const [tlsAllowed, setTlsAllowed] = useState<Record<string, unknown> | undefined>();
  const [tlsHost, setTlsHost] = useState<Record<string, unknown> | undefined>();
  const [fetchingRotation, setFetchingRotation] = useState(false);
  const [fetchingTlsList, setFetchingTlsList] = useState(false);
  const [fetchingTlsHost, setFetchingTlsHost] = useState(false);
  const [rotationError, setRotationError] = useState<Error | undefined>();
  const [tlsListError, setTlsListError] = useState<Error | undefined>();
  const [tlsHostError, setTlsHostError] = useState<Error | undefined>();
  const [hasRotationSnapshot, setHasRotationSnapshot] = useState(false);
  const [hasTlsListSnapshot, setHasTlsListSnapshot] = useState(false);

  const onLoadRotation = useCallback(async () => {
    setFetchingRotation(true);
    setRotationError(undefined);
    try {
      setRotation(await getOpsDomainRotation());
      setHasRotationSnapshot(true);
    } catch (err) {
      setRotationError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setFetchingRotation(false);
    }
  }, []);

  const onLoadTlsList = useCallback(async () => {
    setFetchingTlsList(true);
    setTlsListError(undefined);
    try {
      setTlsAllowed(await getOpsTlsAllowedList());
      setHasTlsListSnapshot(true);
    } catch (err) {
      setTlsListError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setFetchingTlsList(false);
    }
  }, []);

  const onLookupTlsHost = useCallback(async () => {
    const hostname = draftHostname.trim();
    if (!hostname) {
      setTlsHostError(new Error('Hostname is required.'));
      return;
    }
    setFetchingTlsHost(true);
    setTlsHostError(undefined);
    try {
      setTlsHost(await getOpsTlsAllowedHost(hostname));
    } catch (err) {
      setTlsHostError(err instanceof Error ? err : new Error(String(err)));
      setTlsHost(undefined);
    } finally {
      setFetchingTlsHost(false);
    }
  }, [draftHostname]);

  return (
    <OpsDomains
      rotation={rotation}
      tlsAllowed={tlsAllowed}
      tlsHost={tlsHost}
      draftHostname={draftHostname}
      fetchingRotation={fetchingRotation}
      fetchingTlsList={fetchingTlsList}
      fetchingTlsHost={fetchingTlsHost}
      rotationError={rotationError}
      tlsListError={tlsListError}
      tlsHostError={tlsHostError}
      hasRotationSnapshot={hasRotationSnapshot}
      hasTlsListSnapshot={hasTlsListSnapshot}
      onDraftHostnameChange={setDraftHostname}
      onLoadRotation={onLoadRotation}
      onLoadTlsList={onLoadTlsList}
      onLookupTlsHost={onLookupTlsHost}
    />
  );
}
