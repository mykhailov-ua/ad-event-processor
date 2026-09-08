// L3 consent proofs list + signed POST /consent recorder for operator verification.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { getOpsConsentProofs } from '@/api/ops_api';
import { postConsentBody } from '@/api/platform_api';
import type { ConsentRecord } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { buildConsentRecordJson, signConsentHmacHex } from '@/lib/consent_hmac';
import { mutationError } from '@/lib/mutation_audit';

export function useOpsConsentPageWorkspace() {
  const { data, error, fetching, revalidating } = useResource(
    (signal) => getOpsConsentProofs(signal),
    []
  );

  const [draftUserId, setDraftUserId] = useState('');
  const [draftPurposes, setDraftPurposes] = useState('1');
  const [draftSource, setDraftSource] = useState('admin_ui');
  const [draftTimestamp, setDraftTimestamp] = useState('');
  const [draftSigningSecret, setDraftSigningSecret] = useState('');
  const [draftSignatureHex, setDraftSignatureHex] = useState('');
  const [recording, setRecording] = useState(false);
  const [recordError, setRecordError] = useState<Error | undefined>();
  const [recordSuccess, setRecordSuccess] = useState(false);

  const onRecordConsent = useCallback(async () => {
    const userId = draftUserId.trim();
    const source = draftSource.trim();
    const purposes = Number.parseInt(draftPurposes.trim(), 10);
    if (!userId || !source || !Number.isFinite(purposes)) {
      setRecordError(new Error('User ID, purposes, and source are required'));
      setRecordSuccess(false);
      return;
    }

    const body: ConsentRecord = {
      user_id: userId,
      purposes,
      source,
      timestamp: draftTimestamp.trim() || undefined,
    };
    const bodyJson = buildConsentRecordJson(body);

    setRecording(true);
    setRecordError(undefined);
    setRecordSuccess(false);
    try {
      const secret = draftSigningSecret.trim();
      const manualSig = draftSignatureHex.trim();
      let signature = manualSig;
      if (secret) {
        signature = await signConsentHmacHex(secret, bodyJson);
      }
      if (!signature) {
        throw new Error('Provide an HMAC secret or X-Consent-Signature hex value');
      }
      await postConsentBody(bodyJson, signature);
      setRecordSuccess(true);
      toast.success('Consent recorded');
      setDraftUserId('');
      setDraftTimestamp('');
      setDraftSignatureHex('');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setRecordError(nextError);
      toast.error(nextError.message);
    } finally {
      setRecording(false);
    }
  }, [
    draftPurposes,
    draftSignatureHex,
    draftSigningSecret,
    draftSource,
    draftTimestamp,
    draftUserId,
  ]);

  return {
    payload: data,
    fetching,
    listRevalidating: revalidating,
    error,
    hasSnapshot: data != null,
    draftUserId,
    draftPurposes,
    draftSource,
    draftTimestamp,
    draftSigningSecret,
    draftSignatureHex,
    recording,
    recordError,
    recordSuccess,
    onDraftUserIdChange: setDraftUserId,
    onDraftPurposesChange: setDraftPurposes,
    onDraftSourceChange: setDraftSource,
    onDraftTimestampChange: setDraftTimestamp,
    onDraftSigningSecretChange: setDraftSigningSecret,
    onDraftSignatureHexChange: setDraftSignatureHex,
    onRecordConsent: () => {
      void onRecordConsent();
    },
  };
}
