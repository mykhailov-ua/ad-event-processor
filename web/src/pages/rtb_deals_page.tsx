import { useCallback, useEffect, useState } from 'react';
import {
  createRtbDeal,
  deleteRtbDeal,
  fetchRtbDeals,
  patchRtbDeal,
  type RtbDeal,
  type RtbDealCreateSpec,
} from '../helpers/rtb_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { DealsDirectory } from '../ui/rtb/deals_directory.js';

export function RtbDealsPage() {
  const [items, setItems] = useState<RtbDeal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);

  const [modalOpen, setModalOpen] = useState(false);
  const [modalMode, setModalMode] = useState<'create' | 'edit'>('create');
  const [editingDeal, setEditingDeal] = useState<RtbDeal | null>(null);
  const [modalBusy, setModalBusy] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchRtbDeals(ctrl.signal));
      if (cancelled) return;
      if (err) {
        if (err.name === 'AbortError') return;
        setError(err);
        setLoading(false);
        return;
      }
      setItems(result ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  const onOpenCreate = useCallback(() => {
    setModalMode('create');
    setEditingDeal(null);
    setModalError(null);
    setModalOpen(true);
  }, []);

  const onOpenEdit = useCallback((deal: RtbDeal) => {
    setModalMode('edit');
    setEditingDeal(deal);
    setModalError(null);
    setModalOpen(true);
  }, []);

  const onCloseModal = useCallback(() => {
    if (modalBusy) return;
    setModalOpen(false);
    setEditingDeal(null);
    setModalError(null);
  }, [modalBusy]);

  const onSubmitModal = useCallback(
    async (body: RtbDealCreateSpec) => {
      setModalBusy(true);
      setModalError(null);
      try {
        if (modalMode === 'create') {
          await createRtbDeal(body);
          pushToastMessage({ title: 'Deal created', message: body.deal_id });
        } else if (editingDeal?.id != null) {
          await patchRtbDeal(editingDeal.id, body);
          pushToastMessage({ title: 'Deal updated', message: body.deal_id });
        }
        setModalOpen(false);
        setEditingDeal(null);
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        setModalError(err instanceof Error ? err.message : 'Save failed');
      } finally {
        setModalBusy(false);
      }
    },
    [modalMode, editingDeal, reload]
  );

  const onDelete = useCallback(
    async (deal: RtbDeal) => {
      if (deal.id == null) return;
      try {
        await deleteRtbDeal(deal.id);
        pushToastMessage({ title: 'Deal deleted', message: deal.deal_id ?? String(deal.id) });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        setError(err);
      }
    },
    [reload]
  );

  return (
    <DealsDirectory
      items={items}
      loading={loading}
      error={error}
      modalOpen={modalOpen}
      modalMode={modalMode}
      editingDeal={editingDeal}
      modalBusy={modalBusy}
      modalError={modalError}
      onOpenCreate={onOpenCreate}
      onOpenEdit={onOpenEdit}
      onCloseModal={onCloseModal}
      onSubmitModal={(body) => void onSubmitModal(body)}
      onDelete={(deal) => void onDelete(deal)}
    />
  );
}
