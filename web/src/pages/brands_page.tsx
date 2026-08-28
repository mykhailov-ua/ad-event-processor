import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { createBrand, fetchBrands, type Brand } from '../helpers/brands_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { BrandsDirectory } from '../ui/brands/brands_directory.js';

export function BrandsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';

  const [items, setItems] = useState<Brand[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [expandedBrandId, setExpandedBrandId] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [modalBusy, setModalBusy] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    if (!customerId) {
      setItems([]);
      setLoading(false);
      setError(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchBrands(customerId, ctrl.signal));
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
  }, [customerId, reloadToken]);

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextCustomerId) {
        next.set('customer_id', nextCustomerId);
      } else {
        next.delete('customer_id');
      }
      setSearchParams(next, { replace: true });
      setExpandedBrandId(null);
    },
    [searchParams, setSearchParams]
  );

  const onToggleExpand = useCallback((brandId: string) => {
    setExpandedBrandId((prev) => (prev === brandId ? null : brandId));
  }, []);

  const onOpenCreate = useCallback(() => {
    setModalError(null);
    setModalOpen(true);
  }, []);

  const onCloseModal = useCallback(() => {
    if (modalBusy) return;
    setModalOpen(false);
    setModalError(null);
  }, [modalBusy]);

  const onSubmitCreate = useCallback(
    async (body: { customer_id: string; name: string }) => {
      setModalBusy(true);
      setModalError(null);
      try {
        await createBrand(body);
        pushToastMessage({ title: 'Brand created', message: body.name });
        setModalOpen(false);
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        setModalError(err instanceof Error ? err.message : 'Create failed');
      } finally {
        setModalBusy(false);
      }
    },
    [reload]
  );

  return (
    <BrandsDirectory
      customerId={customerId}
      items={items}
      loading={loading}
      error={error}
      expandedBrandId={expandedBrandId}
      modalOpen={modalOpen}
      modalBusy={modalBusy}
      modalError={modalError}
      onCustomerApply={onCustomerApply}
      onToggleExpand={onToggleExpand}
      onOpenCreate={onOpenCreate}
      onCloseModal={onCloseModal}
      onSubmitCreate={(body) => void onSubmitCreate(body)}
      onReload={reload}
    />
  );
}
