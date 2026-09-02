import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import {
  applyPlatformSettings,
  bootstrapPlatformSettings,
  getPlatformSettings,
  patchPlatformSettings,
} from '@/api/settings_api';
import { PlatformSettings } from '@/domains/settings/platform_settings';
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

export function SettingsPage() {
  const [refreshToken, setRefreshToken] = useState(0);
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
    [refreshToken],
  );

  const payload = data as Record<string, unknown> | undefined;

  const onApplyPatch = useCallback(async () => {
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
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setPatchError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setPatching(false);
    }
  }, [draftPatchJson]);

  const onPatchPlatform = useCallback(async (patch: Record<string, unknown>) => {
    setPatching(true);
    setPatchError(undefined);
    setPatchSuccess(false);
    try {
      await patchPlatformSettings(patch);
      setPatchSuccess(true);
      toast.success('Platform settings updated');
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setPatchError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setPatching(false);
    }
  }, []);

  const onApplyToDisk = useCallback(async () => {
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
    } catch (err) {
      setApplyError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [draftInstallRoot]);

  const onRunBootstrap = useCallback(async () => {
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
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setBootstrapError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setBootstrapping(false);
    }
  }, [draftInstallToken, draftBootstrapJson]);

  return (
    <PlatformSettings
      payload={payload}
      draftPatchJson={draftPatchJson}
      draftInstallRoot={draftInstallRoot}
      draftInstallToken={draftInstallToken}
      draftBootstrapJson={draftBootstrapJson}
      fetching={fetching}
      patching={patching}
      applying={applying}
      bootstrapping={bootstrapping}
      error={error}
      patchError={patchError}
      applyError={applyError}
      bootstrapError={bootstrapError}
      patchSuccess={patchSuccess}
      applySuccess={applySuccess}
      bootstrapSuccess={bootstrapSuccess}
      applyWrittenPath={applyWrittenPath}
      hasSnapshot={data != null}
      restartRequired={readRestartRequired(payload)}
      showBootstrap={!readBootstrapComplete(payload)}
      onDraftPatchJsonChange={setDraftPatchJson}
      onDraftInstallRootChange={setDraftInstallRoot}
      onDraftInstallTokenChange={setDraftInstallToken}
      onDraftBootstrapJsonChange={setDraftBootstrapJson}
      onApplyPatch={onApplyPatch}
      onPatchPlatform={onPatchPlatform}
      onApplyToDisk={onApplyToDisk}
      onRunBootstrap={onRunBootstrap}
    />
  );
}
