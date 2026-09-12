import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import { listFraudPresets, patchFraudPreset } from '@/api/fraud_api';
import type { PatchFraudPolicyPresetRequest } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { toError } from '@/lib/admin_error';
import { sessionHasPermission } from '@/lib/session_permissions';
import { useSession } from '@/hooks/use_session';

function parseThreshold(raw: string): number | undefined {
  const trimmed = raw.trim();
  if (trimmed === '') {
    return undefined;
  }
  const value = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(value)) {
    return undefined;
  }
  return value;
}

export function useOpsFraudPresetWorkspace() {
  const { user } = useSession();
  const canList =
    sessionHasPermission(user?.permissions, 'shards:read') ||
    sessionHasPermission(user?.permissions, 'audit:read') ||
    sessionHasPermission(user?.permissions, 'campaigns:read');
  const canPatch = sessionHasPermission(user?.permissions, 'shards:write');

  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching, revalidating } = useResource(
    (signal) => {
      if (!canList) {
        return Promise.resolve(undefined);
      }
      return listFraudPresets(signal);
    },
    [canList, refreshToken]
  );

  const presets = data ?? [];

  const [selectedName, setSelectedName] = useState('');
  const [draftPass, setDraftPass] = useState('');
  const [draftSuspect, setDraftSuspect] = useState('');
  const [draftIvt, setDraftIvt] = useState('');
  const [draftBlock, setDraftBlock] = useState('');
  const [patching, setPatching] = useState(false);
  const [patchError, setPatchError] = useState<Error | undefined>();

  useEffect(() => {
    if (!selectedName && presets.length > 0) {
      const first = presets[0]?.name;
      if (first) {
        setSelectedName(first);
      }
    }
  }, [presets, selectedName]);

  useEffect(() => {
    const preset = presets.find((row) => row.name === selectedName);
    if (!preset) {
      setDraftPass('');
      setDraftSuspect('');
      setDraftIvt('');
      setDraftBlock('');
      return;
    }
    setDraftPass(preset.pass != null ? String(preset.pass) : '');
    setDraftSuspect(preset.suspect != null ? String(preset.suspect) : '');
    setDraftIvt(preset.ivt != null ? String(preset.ivt) : '');
    setDraftBlock(preset.block != null ? String(preset.block) : '');
  }, [presets, selectedName]);

  const onPatchPreset = useCallback(async () => {
    if (patching || !canPatch || !selectedName) {
      return;
    }
    const body: PatchFraudPolicyPresetRequest = {};
    const pass = parseThreshold(draftPass);
    const suspect = parseThreshold(draftSuspect);
    const ivt = parseThreshold(draftIvt);
    const block = parseThreshold(draftBlock);
    if (pass != null) {
      body.pass = pass;
    }
    if (suspect != null) {
      body.suspect = suspect;
    }
    if (ivt != null) {
      body.ivt = ivt;
    }
    if (block != null) {
      body.block = block;
    }
    if (Object.keys(body).length === 0) {
      setPatchError(new Error('Enter at least one threshold to patch'));
      return;
    }
    setPatching(true);
    setPatchError(undefined);
    try {
      const updated = await patchFraudPreset(selectedName, body);
      if (updated.pass != null) {
        setDraftPass(String(updated.pass));
      }
      if (updated.suspect != null) {
        setDraftSuspect(String(updated.suspect));
      }
      if (updated.ivt != null) {
        setDraftIvt(String(updated.ivt));
      }
      if (updated.block != null) {
        setDraftBlock(String(updated.block));
      }
      bumpRefresh();
      toast.success('Fraud preset updated');
    } catch (err) {
      setPatchError(toError(err));
    } finally {
      setPatching(false);
    }
  }, [
    bumpRefresh,
    canPatch,
    draftBlock,
    draftIvt,
    draftPass,
    draftSuspect,
    patching,
    selectedName,
  ]);

  return {
    canList,
    canPatch,
    presets,
    fetching,
    revalidating,
    listError: error,
    hasSnapshot: data != null,
    selectedName,
    setSelectedName,
    draftPass,
    setDraftPass,
    draftSuspect,
    setDraftSuspect,
    draftIvt,
    setDraftIvt,
    draftBlock,
    setDraftBlock,
    patching,
    patchError,
    onPatchPreset,
  };
}

export type OpsFraudPresetWorkspace = ReturnType<typeof useOpsFraudPresetWorkspace>;
