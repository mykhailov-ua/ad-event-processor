// L3 platform settings: GET snapshot + bootstrap/apply/patch mutations; restart_required flag surfacing.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import {
  applyPlatformSettings,
  bootstrapPlatformSettings,
  getPlatformSettings,
  patchPlatformSettings,
} from '@/api/settings_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

function readBootstrapComplete(payload: Record<string, unknown> | undefined): boolean {
  const value = payload?.bootstrap_complete;
  return value === true || value === 'true';
}

function readRestartRequired(payload: Record<string, unknown> | undefined): boolean {
  if (!payload) {
    return false;
  }
  const value = payload.restart_required;
  if (value === true || value === 'true') {
    return true;
  }
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  return false;
}

export function useSettingsPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [draftPatchJson, setDraftPatchJson] = useState('');
  const [draftInstallRoot, setDraftInstallRoot] = useState('');
  const [draftInstallToken, setDraftInstallToken] = useState('');
  const [draftBootstrapJson, setDraftBootstrapJson] = useState('');
  const [patching, setPatching] = useState(false);
  const [applying, setApplying] = useState(false);
  const [bootstrapping, setBootstrapping] = useState(false);
  const [patchError, setPatchError] = useState<Error | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [bootstrapError, setBootstrapError] = useState<Error | undefined>();
  const [patchSuccess, setPatchSuccess] = useState(false);
  const [applySuccess, setApplySuccess] = useState(false);
  const [bootstrapSuccess, setBootstrapSuccess] = useState(false);
  const [applyWrittenPath, setApplyWrittenPath] = useState<string | undefined>();

  const { data, error, fetching } = useResource(
    (signal) => getPlatformSettings(signal),
    [refreshToken]
  );

  const payload = data as Record<string, unknown> | undefined;

  const settingsBusy = fetching || patching || applying || bootstrapping;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, settingsBusy);

  const onApplyPatch = useCallback(async () => {
    if (patching) {
      return;
    }
    const trimmed = draftPatchJson.trim();
    if (!trimmed) {
      return;
    }
    setPatching(true);
    setPatchError(undefined);
    setPatchSuccess(false);
    try {
      const parsed: unknown = JSON.parse(trimmed);
      if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error('Patch must be a JSON object');
      }
      await patchPlatformSettings(parsed as Record<string, unknown>);
      setPatchSuccess(true);
      toast.success('Platform settings updated');
      setDraftPatchJson('');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setPatchError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setPatching(false);
    }
  }, [draftPatchJson, bumpRefreshCoalesced, patching]);

  const onPatchPlatform = useCallback(
    async (patch: Record<string, unknown>) => {
      if (patching) {
        return;
      }
      setPatching(true);
      setPatchError(undefined);
      setPatchSuccess(false);
      try {
        await patchPlatformSettings(patch);
        setPatchSuccess(true);
        toast.success('Platform settings updated');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const patchFailure = err instanceof Error ? err : new Error(String(err));
        setPatchError(patchFailure);
        toast.error(patchFailure.message);
      } finally {
        setPatching(false);
      }
    },
    [bumpRefreshCoalesced, patching]
  );

  const onApplyToDisk = useCallback(async () => {
    if (applying) {
      return;
    }
    setApplying(true);
    setApplyError(undefined);
    setApplySuccess(false);
    setApplyWrittenPath(undefined);
    try {
      const trimmed = draftInstallRoot.trim();
      const response = await applyPlatformSettings(trimmed ? { install_root: trimmed } : undefined);
      setApplySuccess(true);
      toast.success('Configuration saved to disk');
      setApplyWrittenPath(response.written_path);
    } catch (err: unknown) {
      setApplyError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [applying, draftInstallRoot]);

  const onRunBootstrap = useCallback(async () => {
    if (bootstrapping) {
      return;
    }
    const token = draftInstallToken.trim();
    const trimmed = draftBootstrapJson.trim();
    if (!token || !trimmed) {
      return;
    }
    setBootstrapping(true);
    setBootstrapError(undefined);
    setBootstrapSuccess(false);
    try {
      const parsed: unknown = JSON.parse(trimmed);
      if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error('Bootstrap body must be a JSON object');
      }
      await bootstrapPlatformSettings(token, parsed as Record<string, unknown>);
      setBootstrapSuccess(true);
      toast.success('Initial setup complete');
      setDraftInstallToken('');
      setDraftBootstrapJson('');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setBootstrapError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setBootstrapping(false);
    }
  }, [bootstrapping, draftInstallToken, draftBootstrapJson, bumpRefreshCoalesced]);

  return {
    payload,
    draftPatchJson,
    draftInstallRoot,
    draftInstallToken,
    draftBootstrapJson,
    fetching,
    patching,
    applying,
    bootstrapping,
    error,
    patchError,
    applyError,
    bootstrapError,
    patchSuccess,
    applySuccess,
    bootstrapSuccess,
    applyWrittenPath,
    hasSnapshot: data != null,
    restartRequired: readRestartRequired(payload),
    showBootstrap: !readBootstrapComplete(payload),
    onDraftPatchJsonChange: setDraftPatchJson,
    onDraftInstallRootChange: setDraftInstallRoot,
    onDraftInstallTokenChange: setDraftInstallToken,
    onDraftBootstrapJsonChange: setDraftBootstrapJson,
    onApplyPatch,
    onPatchPlatform,
    onApplyToDisk,
    onRunBootstrap,
  };
}
