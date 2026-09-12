import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  applyAccessRolesYaml,
  getAccessCatalog,
  getAccessRolesYaml,
  validateAccessRolesYaml,
} from '@/api/access_api';
import { useResource } from '@/api/use_resource';
import { useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useSession } from '@/hooks/use_session';
import { toError, userErrorMessage } from '@/lib/admin_error';
import { sessionHasPermission } from '@/lib/session_permissions';

export function useAccessRolesWorkspace() {
  const { user, refetchSession } = useSession();
  const permissions = user?.permissions;
  const canRead = sessionHasPermission(permissions, 'access:read');
  const canWrite = sessionHasPermission(permissions, 'access:write');

  const {
    data: catalog,
    error: catalogError,
    fetching: catalogFetching,
  } = useResource(
    (signal) => (canRead ? getAccessCatalog(signal) : Promise.resolve(undefined)),
    [canRead]
  );

  const { refreshToken: yamlRefreshToken, bumpRefresh: bumpYamlRefresh } = useRefreshToken();
  const {
    data: yamlText,
    error: yamlError,
    fetching: yamlFetching,
    revalidating: yamlRevalidating,
  } = useResource(
    (signal) => (canRead ? getAccessRolesYaml(signal) : Promise.resolve('')),
    [canRead, yamlRefreshToken]
  );

  const [draftYaml, setDraftYaml] = useState('');
  const [draftRevision, setDraftRevision] = useState(0);
  const [validating, setValidating] = useState(false);
  const [applying, setApplying] = useState(false);
  const [validationError, setValidationError] = useState<Error | undefined>();
  const [actionError, setActionError] = useState<Error | undefined>();

  const syncDraftFromServer = useCallback(() => {
    if (yamlText) {
      setDraftYaml(yamlText);
      const match = yamlText.match(/^revision:\s*(\d+)/m);
      setDraftRevision(match ? Number(match[1]) : 0);
    }
  }, [yamlText]);

  useEffect(() => {
    if (draftYaml === '' && yamlText) {
      syncDraftFromServer();
    }
  }, [draftYaml, syncDraftFromServer, yamlText]);

  const onValidate = useCallback(async () => {
    if (!canWrite || !draftYaml.trim()) {
      return;
    }
    setValidating(true);
    setValidationError(undefined);
    try {
      const result = await validateAccessRolesYaml(draftYaml);
      if (!result.valid) {
        setValidationError(
          new Error(result.errors?.map((row) => row.code).join(', ') || 'invalid')
        );
        return;
      }
      toast.success('Validation passed');
    } catch (error) {
      setValidationError(toError(error));
    } finally {
      setValidating(false);
    }
  }, [canWrite, draftYaml]);

  const onApply = useCallback(async () => {
    if (!canWrite || !draftYaml.trim()) {
      return;
    }
    setApplying(true);
    setActionError(undefined);
    try {
      const result = await applyAccessRolesYaml(draftYaml, draftRevision);
      setDraftRevision(result.revision);
      toast.success('Roles applied');
      bumpYamlRefresh();
      await refetchSession();
    } catch (error) {
      setActionError(toError(error));
      toast.error(userErrorMessage(error));
    } finally {
      setApplying(false);
    }
  }, [bumpYamlRefresh, canWrite, draftRevision, draftYaml, refetchSession]);

  return {
    canRead,
    canWrite,
    catalog,
    catalogError,
    catalogFetching,
    yamlError,
    yamlFetching: yamlFetching || yamlRevalidating,
    draftYaml,
    onDraftYamlChange: setDraftYaml,
    onValidate,
    onApply,
    validating,
    applying,
    validationError,
    actionError,
    syncDraftFromServer,
  };
}
