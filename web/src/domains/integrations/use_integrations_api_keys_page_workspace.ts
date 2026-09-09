import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  createSelfServeApiKey,
  listSelfServeApiKeys,
  revokeSelfServeApiKey,
} from '@/api/selfserve_api';
import type { APIKeyCreatedResponse, APIKeySummary } from '@/api/types';
import { mutationError } from '@/lib/mutation_audit';
import {
  requireNonEmpty,
  toastValidationError,
  validationError,
  type AdminValidationError,
} from '@/lib/admin_validation_error';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export const SELF_SERVE_API_KEY_SCOPES = [
  'campaigns:read',
  'campaigns:read:masked',
  'campaigns:write',
  'campaigns:pause',
  'customers:read',
] as const;

export function useIntegrationsApiKeysPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching, revalidating } = useResource(
    (signal) => listSelfServeApiKeys(signal),
    [refreshToken]
  );

  const [draftName, setDraftName] = useState('');
  const [draftScopes, setDraftScopes] = useState<string[]>(['campaigns:read']);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [formValidationError, setFormValidationError] = useState<AdminValidationError | undefined>();
  const [createdKey, setCreatedKey] = useState<APIKeyCreatedResponse | undefined>();
  const [revokingId, setRevokingId] = useState<string | undefined>();
  const [revokeError, setRevokeError] = useState<Error | undefined>();

  const keys = useMemo(() => data?.keys ?? [], [data?.keys]);
  const listBusy = fetching || creating || revokingId != null;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const onToggleScope = useCallback((scope: string, checked: boolean) => {
    setDraftScopes((current) => {
      if (checked) {
        if (current.includes(scope)) {
          return current;
        }
        return [...current, scope];
      }
      return current.filter((value) => value !== scope);
    });
  }, []);

  const onCreate = useCallback(async () => {
    if (creating) {
      return;
    }
    const nameCheck = requireNonEmpty(draftName, 'Key name', 'name');
    if (!nameCheck.ok) {
      setFormValidationError(nameCheck.error);
      toastValidationError(nameCheck.error);
      return;
    }
    if (draftScopes.length === 0) {
      const err = validationError('Select at least one scope.', { field: 'scopes' });
      setFormValidationError(err);
      toastValidationError(err);
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setFormValidationError(undefined);
    setCreatedKey(undefined);
    try {
      const response = await createSelfServeApiKey({
        name: nameCheck.value,
        scopes: draftScopes,
      });
      setCreatedKey(response);
      setDraftName('');
      toast.success('API key created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCreateError(mutationError(err));
    } finally {
      setCreating(false);
    }
  }, [bumpRefreshCoalesced, creating, draftName, draftScopes]);

  const onRevoke = useCallback(
    async (row: APIKeySummary) => {
      const keyId = row.id?.trim();
      if (!keyId || revokingId != null) {
        return;
      }
      setRevokingId(keyId);
      setRevokeError(undefined);
      try {
        await revokeSelfServeApiKey(keyId);
        toast.success('API key revoked');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        setRevokeError(mutationError(err));
      } finally {
        setRevokingId(undefined);
      }
    },
    [bumpRefreshCoalesced, revokingId]
  );

  const onDismissCreatedKey = useCallback(() => {
    setCreatedKey(undefined);
  }, []);

  return {
    keys,
    fetching,
    listRevalidating: revalidating,
    error,
    hasSnapshot: data != null,
    draftName,
    draftScopes,
    creating,
    createError,
    formValidationError,
    createdKey,
    revokingId,
    revokeError,
    onDraftNameChange: (value: string) => {
      setFormValidationError(undefined);
      setDraftName(value);
    },
    onToggleScope,
    onCreate,
    onRevoke,
    onDismissCreatedKey,
  };
}
