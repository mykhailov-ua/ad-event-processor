import { useCallback, useEffect, useState } from 'react';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  createSupplyAdsTxt,
  createSupplySeller,
  deleteSupplyAdsTxt,
  deleteSupplySeller,
  fetchSupplyAdsTxt,
  fetchSupplyAdsTxtPreview,
  fetchSupplyExportPath,
  fetchSupplySellers,
  fetchSupplySellersPreview,
  fetchSupplyValidation,
  type SupplyAdsTxtEntry,
  type SupplyExportPath,
  type SupplySeller,
  type SupplyValidation,
} from '../helpers/integrations_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { SupplyPanel } from '../ui/supply/supply_panel.js';

export function IntegrationsSupplyPage() {
  const [sellers, setSellers] = useState<SupplySeller[]>([]);
  const [adsTxt, setAdsTxt] = useState<SupplyAdsTxtEntry[]>([]);
  const [validation, setValidation] = useState<SupplyValidation | null>(null);
  const [exportPath, setExportPath] = useState<SupplyExportPath | null>(null);
  const [sellersPreview, setSellersPreview] = useState('');
  const [adsTxtPreview, setAdsTxtPreview] = useState('');
  const [previewTab, setPreviewTab] = useState<'sellers' | 'ads_txt'>('sellers');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [sellersResult, sellersErr] = await to(fetchSupplySellers(ctrl.signal));
      if (cancelled) return;
      if (sellersErr && sellersErr.name !== 'AbortError') {
        setError(sellersErr);
        setLoading(false);
        return;
      }
      setSellers(sellersResult ?? []);

      const [adsResult, adsErr] = await to(fetchSupplyAdsTxt(ctrl.signal));
      if (cancelled) return;
      if (adsErr && adsErr.name !== 'AbortError') {
        setError(adsErr);
        setLoading(false);
        return;
      }
      setAdsTxt(adsResult ?? []);

      const [validationResult] = await to(fetchSupplyValidation(ctrl.signal));
      if (!cancelled) setValidation(validationResult ?? null);

      const [exportResult] = await to(fetchSupplyExportPath(ctrl.signal));
      if (!cancelled) setExportPath(exportResult ?? null);

      const [sellersPreviewResult] = await to(fetchSupplySellersPreview(ctrl.signal));
      if (!cancelled) setSellersPreview(sellersPreviewResult ?? '');

      const [adsPreviewResult] = await to(fetchSupplyAdsTxtPreview(ctrl.signal));
      if (!cancelled) setAdsTxtPreview(adsPreviewResult ?? '');

      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  const onReloadValidation = useCallback(() => {
    reload();
  }, [reload]);

  const onCreateSeller = useCallback(
    async (body: {
      seller_id: string;
      domain: string;
      seller_type: string;
      name: string;
      is_confidential: boolean;
    }) => {
      setBusy(true);
      try {
        await createSupplySeller(body);
        pushToastMessage({ title: 'Seller created', message: body.seller_id });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Create failed',
          message: err instanceof Error ? err.message : 'Create failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onDeleteSeller = useCallback(
    async (id: number) => {
      setBusy(true);
      try {
        await deleteSupplySeller(id);
        pushToastMessage({ title: 'Seller deleted', message: String(id) });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Delete failed',
          message: err instanceof Error ? err.message : 'Delete failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onCreateAdsTxt = useCallback(
    async (body: {
      domain: string;
      publisher_account_id: string;
      relationship: string;
      cert_authority_id?: string;
      sort_order?: number;
    }) => {
      setBusy(true);
      try {
        await createSupplyAdsTxt(body);
        pushToastMessage({ title: 'ads.txt line added', message: body.domain });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Create failed',
          message: err instanceof Error ? err.message : 'Create failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onDeleteAdsTxt = useCallback(
    async (id: number) => {
      setBusy(true);
      try {
        await deleteSupplyAdsTxt(id);
        pushToastMessage({ title: 'ads.txt line deleted', message: String(id) });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Delete failed',
          message: err instanceof Error ? err.message : 'Delete failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  return (
    <SupplyPanel
      sellers={sellers}
      adsTxt={adsTxt}
      validation={validation}
      exportPath={exportPath}
      sellersPreview={sellersPreview}
      adsTxtPreview={adsTxtPreview}
      previewTab={previewTab}
      loading={loading}
      error={error}
      busy={busy}
      onPreviewTabChange={setPreviewTab}
      onReloadValidation={onReloadValidation}
      onCreateSeller={(body) => {
        void onCreateSeller(body);
      }}
      onDeleteSeller={(id) => {
        void onDeleteSeller(id);
      }}
      onCreateAdsTxt={(body) => {
        void onCreateAdsTxt(body);
      }}
      onDeleteAdsTxt={(id) => {
        void onDeleteAdsTxt(id);
      }}
    />
  );
}
