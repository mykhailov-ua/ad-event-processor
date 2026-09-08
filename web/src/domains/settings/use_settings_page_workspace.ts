// L3 platform settings: GET snapshot + bootstrap/apply/patch mutations; restart_required flag surfacing.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import {
  applyPlatformSettings,
  bootstrapPlatformSettings,
  getPlatformSettings,
  patchPlatformSettings,
} from '@/api/settings_api';
import type {
  PlatformBootstrapRequest,
  PlatformSettingsPatch,
  PlatformSettingsView,
} from '@/api/types';
import type { SettingsBootstrapDraft } from '@/domains/settings/settings_bootstrap_form';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';

function readBootstrapComplete(payload: PlatformSettingsView | undefined): boolean {
  return payload?.bootstrap_complete === true;
}

function readRestartRequired(payload: PlatformSettingsView | undefined): boolean {
  if (!payload) {
    return false;
  }
  const value = payload.restart_required;
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  return false;
}

function buildBootstrapBody(draft: SettingsBootstrapDraft): PlatformBootstrapRequest {
  const body: PlatformBootstrapRequest = {
    admin_email: draft.admin_email,
    admin_password: draft.admin_password,
    config: {
      tracking_domain: draft.tracking_domain,
      default_currency: draft.default_currency,
      timezone: draft.timezone,
      ingress_schema: draft.ingress_schema as PlatformBootstrapRequest['config']['ingress_schema'],
      telemetry_enabled: draft.telemetry_enabled,
      edge_xdp: draft.edge_xdp,
      edge_expose_click: draft.edge_expose_click,
      edge_expose_openrtb: draft.edge_expose_openrtb,
      network_interface: draft.network_interface,
    },
  };
  if (draft.license_key) {
    body.license_key = draft.license_key;
  }
  if (draft.license_server) {
    body.license_server = draft.license_server;
  }
  if (draft.deployment_id) {
    body.deployment_id = draft.deployment_id;
  }
  if (draft.eula_version) {
    body.eula_version = draft.eula_version;
  }
  return body;
}

export function useSettingsPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [draftInstallRoot, setDraftInstallRoot] = useState('');
  const [draftInstallToken, setDraftInstallToken] = useState('');
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

  const payload = data as PlatformSettingsView | undefined;

  const settingsBusy = fetching || patching || applying || bootstrapping;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, settingsBusy);

  const onPatchPlatform = useCallback(
    async (patch: PlatformSettingsPatch) => {
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
    if (!confirmDestructiveAction('Write platform configuration to disk on the server?')) {
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
      const nextError = mutationError(err);
      setApplyError(nextError);
      toast.error(nextError.message);
    } finally {
      setApplying(false);
    }
  }, [applying, draftInstallRoot]);

  const onRunBootstrap = useCallback(
    async (draft: SettingsBootstrapDraft) => {
      if (bootstrapping) {
        return;
      }
      const token = draftInstallToken.trim();
      if (!token) {
        return;
      }
      if (
        !confirmDestructiveAction('Run initial platform bootstrap with the provided install token?')
      ) {
        return;
      }
      setBootstrapping(true);
      setBootstrapError(undefined);
      setBootstrapSuccess(false);
      try {
        await bootstrapPlatformSettings(token, buildBootstrapBody(draft));
        setBootstrapSuccess(true);
        toast.success('Initial setup complete');
        setDraftInstallToken('');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setBootstrapError(nextError);
        toast.error(nextError.message);
      } finally {
        setBootstrapping(false);
      }
    },
    [bootstrapping, bumpRefreshCoalesced, draftInstallToken]
  );

  return {
    payload,
    draftInstallRoot,
    draftInstallToken,
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
    onDraftInstallRootChange: setDraftInstallRoot,
    onDraftInstallTokenChange: setDraftInstallToken,
    onPatchPlatform,
    onApplyToDisk,
    onRunBootstrap,
  };
}
