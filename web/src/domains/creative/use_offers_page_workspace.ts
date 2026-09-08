// offers directory: list, create, edit, delete.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createOffer, deleteOffer, listOffers, updateOffer } from '@/api/offers_api';
import type { Offer } from '@/api/types';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useOffersPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource((signal) => listOffers(signal), [refreshToken]);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const [editingOffer, setEditingOffer] = useState<Offer | null>(null);
  const [editName, setEditName] = useState('');
  const [editUrl, setEditUrl] = useState('');
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<Error | undefined>();

  const [actingOfferId, setActingOfferId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<Error | undefined>();

  const onCreateOffer = useCallback(async () => {
    if (creating) {
      return;
    }
    const name = draftName.trim();
    const url = draftUrl.trim();
    if (!name || !url) {
      setCreateError(new Error('Name and URL are required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createOffer({ name, url });
      setCreateSuccess(true);
      setDraftName('');
      setDraftUrl('');
      toast.success('Offer created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [bumpRefreshCoalesced, creating, draftName, draftUrl]);

  const onOpenEditOffer = useCallback((offer: Offer) => {
    setEditingOffer(offer);
    setEditName(offer.name);
    setEditUrl(offer.url);
    setEditError(undefined);
  }, []);

  const onCloseEditOffer = useCallback(() => {
    setEditingOffer(null);
    setEditError(undefined);
  }, []);

  const onSaveOffer = useCallback(async () => {
    if (!editingOffer || saving) {
      return;
    }
    const name = editName.trim();
    const url = editUrl.trim();
    if (!name || !url) {
      setEditError(new Error('Name and URL are required.'));
      return;
    }
    setSaving(true);
    setEditError(undefined);
    try {
      await updateOffer(editingOffer.id, { name, url });
      toast.success('Offer updated');
      bumpRefreshCoalesced();
      setEditingOffer(null);
    } catch (err: unknown) {
      setEditError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [bumpRefreshCoalesced, editName, editUrl, editingOffer, saving]);

  const onDeleteOffer = useCallback(
    async (offer: Offer) => {
      if (actingOfferId) {
        return;
      }
      const confirmed = confirmDestructiveAction(`Delete offer "${offer.name}"?`);
      if (!confirmed) {
        return;
      }
      setActingOfferId(offer.id);
      setActionError(undefined);
      try {
        await deleteOffer(offer.id);
        toast.success('Offer deleted');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setActingOfferId(null);
      }
    },
    [actingOfferId, bumpRefreshCoalesced]
  );

  return {
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
    draftName,
    draftUrl,
    creating,
    createError,
    createSuccess,
    onDraftNameChange: setDraftName,
    onDraftUrlChange: setDraftUrl,
    onCreateOffer: () => {
      void onCreateOffer();
    },
    editingOffer,
    editName,
    editUrl,
    saving,
    editError,
    onOpenEditOffer,
    onCloseEditOffer,
    onEditNameChange: setEditName,
    onEditUrlChange: setEditUrl,
    onSaveOffer: () => {
      void onSaveOffer();
    },
    actingOfferId,
    actionError,
    onDeleteOffer: (offer: Offer) => {
      void onDeleteOffer(offer);
    },
  };
}
