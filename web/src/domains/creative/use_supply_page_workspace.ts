// L3 supply hub: parallel GET bundle (sellers, ads.txt, export path, validation) on mount.
import { useCallback, useState } from 'react';

import {
  fetchSupplyPreviewText,
  getSupplyExportPath,
  getSupplyValidation,
  listSupplyAdsTxt,
  listSupplySellers,
  SUPPLY_PREVIEW_ADS_TXT_PATH,
  SUPPLY_PREVIEW_SELLERS_JSON_PATH,
} from '@/api/supply_api';
import { mutationError } from '@/lib/mutation_audit';
import { useResource } from '@/api/use_resource';

type SupplySnapshot = {
  sellers: Awaited<ReturnType<typeof listSupplySellers>>;
  adsTxt: Awaited<ReturnType<typeof listSupplyAdsTxt>>;
  exportPath: Awaited<ReturnType<typeof getSupplyExportPath>>;
  validation: Awaited<ReturnType<typeof getSupplyValidation>>;
};

export function useSupplyPageWorkspace() {
  const { data, error, fetching } = useResource<SupplySnapshot>(async (signal) => {
    const [sellers, adsTxt, exportPath, validation] = await Promise.all([
      listSupplySellers(signal),
      listSupplyAdsTxt(signal),
      getSupplyExportPath(signal),
      getSupplyValidation(signal),
    ]);
    return { sellers, adsTxt, exportPath, validation };
  }, []);

  const [previewSellersJson, setPreviewSellersJson] = useState<string | undefined>();
  const [previewAdsTxt, setPreviewAdsTxt] = useState<string | undefined>();
  const [previewSellersError, setPreviewSellersError] = useState<Error | undefined>();
  const [previewAdsTxtError, setPreviewAdsTxtError] = useState<Error | undefined>();
  const [loadingSellersPreview, setLoadingSellersPreview] = useState(false);
  const [loadingAdsTxtPreview, setLoadingAdsTxtPreview] = useState(false);

  const onLoadSellersPreview = useCallback(async () => {
    if (loadingSellersPreview) {
      return;
    }
    setLoadingSellersPreview(true);
    setPreviewSellersError(undefined);
    try {
      const text = await fetchSupplyPreviewText(SUPPLY_PREVIEW_SELLERS_JSON_PATH);
      setPreviewSellersJson(text);
    } catch (err: unknown) {
      setPreviewSellersJson(undefined);
      setPreviewSellersError(mutationError(err));
    } finally {
      setLoadingSellersPreview(false);
    }
  }, [loadingSellersPreview]);

  const onLoadAdsTxtPreview = useCallback(async () => {
    if (loadingAdsTxtPreview) {
      return;
    }
    setLoadingAdsTxtPreview(true);
    setPreviewAdsTxtError(undefined);
    try {
      const text = await fetchSupplyPreviewText(SUPPLY_PREVIEW_ADS_TXT_PATH);
      setPreviewAdsTxt(text);
    } catch (err: unknown) {
      setPreviewAdsTxt(undefined);
      setPreviewAdsTxtError(mutationError(err));
    } finally {
      setLoadingAdsTxtPreview(false);
    }
  }, [loadingAdsTxtPreview]);

  return {
    sellers: data?.sellers ?? [],
    adsTxt: data?.adsTxt ?? [],
    exportPath: data?.exportPath,
    validation: data?.validation,
    fetching,
    error,
    hasSnapshot: data != null,
    previewSellersJson,
    previewAdsTxt,
    previewSellersError,
    previewAdsTxtError,
    loadingSellersPreview,
    loadingAdsTxtPreview,
    onLoadSellersPreview: () => {
      void onLoadSellersPreview();
    },
    onLoadAdsTxtPreview: () => {
      void onLoadAdsTxtPreview();
    },
  };
}
