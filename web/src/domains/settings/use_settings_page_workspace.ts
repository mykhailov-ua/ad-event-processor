// Settings index: platform GET/PATCH/apply + license apply; single draft owner.
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  applyPlatformSettings,
  getPlatformSettings,
  patchPlatformSettings,
} from '@/api/settings_api';
import type { PlatformSettingsView } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useLicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';
import {
  buildPlatformPatchBody,
  configToDraft,
  draftEqualsConfig,
  type PlatformSettingsDraft,
} from '@/domains/settings/settings_draft';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useMeta } from '@/hooks/use_meta';
import { useSession } from '@/hooks/use_session';
import { toError, userErrorMessage } from '@/lib/admin_error';
import { licenseStateLabel } from '@/lib/install_meta';
import { confirmDestructiveAction } from '@/lib/mutation_audit';
import { sessionHasPermission } from '@/lib/session_permissions';

export type SettingsPageWorkspace = {
  meta: ReturnType<typeof useMeta>['meta'];
  metaError: Error | undefined;
  platformSnapshot: PlatformSettingsView | undefined;
  draft: PlatformSettingsDraft;
  draftDirty: boolean;
  canWrite: boolean;
  loading: boolean;
  revalidating: boolean;
  loadError: Error | undefined;
  patchError: Error | undefined;
  applyError: Error | undefined;
  patching: boolean;
  applying: boolean;
  applyWrittenPath: string | undefined;
  installRoot: string;
  onDraftChange: (patch: Partial<PlatformSettingsDraft>) => void;
  onInstallRootChange: (value: string) => void;
  onSave: () => Promise<void>;
  onDiscard: () => void;
  onApply: () => Promise<void>;
  onRefresh: () => void;
  licenseLoad: ReturnType<typeof useLicenseApplyFormLoad>;
  stateLabel: string;
  onLicenseApplied: () => void;
};

export function useSettingsPageWorkspace(): SettingsPageWorkspace {
  const { meta, error: metaError, refreshMeta } = useMeta();
  const { user } = useSession();
  const canWrite = sessionHasPermission(user?.permissions, 'settings:write');
  const licenseLoad = useLicenseApplyFormLoad(false);
  const stateLabel = licenseStateLabel(meta);

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const {
    data: platformSnapshot,
    error: loadError,
    fetching,
    revalidating,
  } = useResource((signal) => getPlatformSettings(signal), [refreshToken]);

  const [draft, setDraft] = useState<PlatformSettingsDraft>(() => configToDraft(undefined));
  const [patching, setPatching] = useState(false);
  const [applying, setApplying] = useState(false);
  const [patchError, setPatchError] = useState<Error | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [applyWrittenPath, setApplyWrittenPath] = useState<string | undefined>();
  const [installRoot, setInstallRoot] = useState('');
  const [hasLocalDraft, setHasLocalDraft] = useState(false);

  const syncDraftFromSnapshot = useCallback((snapshot: PlatformSettingsView | undefined) => {
    setDraft(configToDraft(snapshot?.config));
    setHasLocalDraft(false);
  }, []);

  useEffect(() => {
    if (platformSnapshot && !hasLocalDraft) {
      setDraft(configToDraft(platformSnapshot.config));
    }
  }, [hasLocalDraft, platformSnapshot]);

  const draftDirty = useMemo(
    () => !draftEqualsConfig(draft, platformSnapshot?.config),
    [draft, platformSnapshot?.config]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(
    bumpRefresh,
    fetching || patching || applying
  );

  const onDraftChange = useCallback((patch: Partial<PlatformSettingsDraft>) => {
    setHasLocalDraft(true);
    setDraft((current) => ({ ...current, ...patch }));
    setPatchError(undefined);
  }, []);

  const onInstallRootChange = useCallback((value: string) => {
    setInstallRoot(value);
    setApplyError(undefined);
  }, []);

  const onDiscard = useCallback(() => {
    if (patching) {
      return;
    }
    syncDraftFromSnapshot(platformSnapshot);
    setPatchError(undefined);
  }, [patching, platformSnapshot, syncDraftFromSnapshot]);

  const onRefresh = useCallback(() => {
    bumpRefreshCoalesced();
  }, [bumpRefreshCoalesced]);

  const onSave = useCallback(async () => {
    if (!canWrite || patching || !platformSnapshot) {
      return;
    }
    const patchBody = buildPlatformPatchBody(draft, platformSnapshot.config);
    if (!patchBody) {
      return;
    }
    setPatching(true);
    setPatchError(undefined);
    try {
      const updated = await patchPlatformSettings(patchBody);
      syncDraftFromSnapshot(updated);
      toast.success('Settings saved');
      bumpRefreshCoalesced();
      void refreshMeta();
    } catch (error) {
      setPatchError(toError(error));
      toast.error(userErrorMessage(error));
    } finally {
      setPatching(false);
    }
  }, [
    bumpRefreshCoalesced,
    canWrite,
    draft,
    patching,
    platformSnapshot,
    refreshMeta,
    syncDraftFromSnapshot,
  ]);

  const onApply = useCallback(async () => {
    if (!canWrite || applying) {
      return;
    }
    if (
      !confirmDestructiveAction(
        'Overwrite install.compose.env on the server? This does not restart services.'
      )
    ) {
      return;
    }
    setApplying(true);
    setApplyError(undefined);
    try {
      const trimmedRoot = installRoot.trim();
      const response = await applyPlatformSettings(
        trimmedRoot ? { install_root: trimmedRoot } : undefined
      );
      setApplyWrittenPath(response.written_path);
      toast.success(`Written to ${response.written_path}`);
      bumpRefreshCoalesced();
    } catch (error) {
      setApplyError(toError(error));
      toast.error(userErrorMessage(error));
    } finally {
      setApplying(false);
    }
  }, [applying, bumpRefreshCoalesced, canWrite, installRoot]);

  const onLicenseApplied = useCallback(() => {
    void refreshMeta();
    bumpRefreshCoalesced();
  }, [bumpRefreshCoalesced, refreshMeta]);

  useEffect(() => {
    refreshMeta();
  }, [refreshMeta]);

  return {
    meta,
    metaError,
    platformSnapshot,
    draft,
    draftDirty,
    canWrite,
    loading: fetching && !platformSnapshot && !loadError,
    revalidating,
    loadError,
    patchError,
    applyError,
    patching,
    applying,
    applyWrittenPath,
    installRoot,
    onDraftChange,
    onInstallRootChange,
    onSave,
    onDiscard,
    onApply,
    onRefresh,
    licenseLoad,
    stateLabel,
    onLicenseApplied,
  };
}
