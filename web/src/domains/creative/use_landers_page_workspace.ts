// L3 landers directory: list, refresh band, create/edit/delete, row actions (EH-SI1).
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createLander, deleteLander, listLanders, updateLander } from '@/api/landers_api';
import type { Lander } from '@/api/types';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useLandersPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching, revalidating, updatedAt } = useResource(
    async (signal) => {
      const items = await listLanders(signal);
      return items;
    },
    [refreshToken]
  );

  const listBusy = fetching || revalidating;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const onRefresh = useCallback(() => {
    bumpRefreshCoalesced();
  }, [bumpRefreshCoalesced]);

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const [editingLander, setEditingLander] = useState<Lander | null>(null);
  const [editName, setEditName] = useState('');
  const [editUrl, setEditUrl] = useState('');
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<Error | undefined>();

  const [actingLanderId, setActingLanderId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<Error | undefined>();

  const onCreateLander = useCallback(async () => {
    if (creating) {
      return;
    }
    const name = draftName.trim();
    if (!name) {
      setCreateError(new Error('Lander name is required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createLander({ name, url: draftUrl.trim() || undefined });
      setCreateSuccess(true);
      setDraftName('');
      setDraftUrl('');
      toast.success('Lander created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [bumpRefreshCoalesced, creating, draftName, draftUrl]);

  const onOpenEditLander = useCallback((lander: Lander) => {
    setEditingLander(lander);
    setEditName(lander.name);
    setEditUrl(lander.url?.trim() ?? '');
    setEditError(undefined);
  }, []);

  const onCloseEditLander = useCallback(() => {
    setEditingLander(null);
    setEditError(undefined);
  }, []);

  const onSaveLander = useCallback(async () => {
    if (!editingLander || saving) {
      return;
    }
    const name = editName.trim();
    if (!name) {
      setEditError(new Error('Lander name is required.'));
      return;
    }
    setSaving(true);
    setEditError(undefined);
    try {
      await updateLander(editingLander.id, {
        name,
        url: editingLander.hosted_asset_id ? undefined : editUrl.trim(),
      });
      toast.success('Lander updated');
      bumpRefreshCoalesced();
      setEditingLander(null);
    } catch (err: unknown) {
      setEditError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [bumpRefreshCoalesced, editName, editUrl, editingLander, saving]);

  const onDeleteLander = useCallback(
    async (lander: Lander) => {
      if (actingLanderId) {
        return;
      }
      const confirmed = confirmDestructiveAction(`Delete lander "${lander.name}"?`);
      if (!confirmed) {
        return;
      }
      setActingLanderId(lander.id);
      setActionError(undefined);
      try {
        await deleteLander(lander.id);
        toast.success('Lander deleted');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setActingLanderId(null);
      }
    },
    [actingLanderId, bumpRefreshCoalesced]
  );

  const onCopyUrl = useCallback(async (url: string) => {
    const trimmed = url.trim();
    if (!trimmed) {
      return;
    }
    try {
      await navigator.clipboard.writeText(trimmed);
      toast.success('URL copied');
    } catch {
      toast.error('Could not copy to clipboard');
    }
  }, []);

  return {
    items: data,
    fetching,
    listRevalidating: revalidating,
    error,
    hasSnapshot: data != null,
    listLastUpdatedAt: updatedAt,
    onRefresh,
    draftName,
    draftUrl,
    creating,
    createError,
    createSuccess,
    onDraftNameChange: setDraftName,
    onDraftUrlChange: setDraftUrl,
    onCreateLander: () => {
      void onCreateLander();
    },
    editingLander,
    editName,
    editUrl,
    saving,
    editError,
    onOpenEditLander,
    onCloseEditLander,
    onEditNameChange: setEditName,
    onEditUrlChange: setEditUrl,
    onSaveLander: () => {
      void onSaveLander();
    },
    actingLanderId,
    actionError,
    onDeleteLander: (lander: Lander) => {
      void onDeleteLander(lander);
    },
    onCopyUrl: (url: string) => {
      void onCopyUrl(url);
    },
  };
}
