// L3 supply hub: parallel GET bundle (sellers, ads.txt, export path, validation) on mount.
import {
  getSupplyExportPath,
  getSupplyValidation,
  listSupplyAdsTxt,
  listSupplySellers,
} from '@/api/supply_api';
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

  return {
    sellers: data?.sellers ?? [],
    adsTxt: data?.adsTxt ?? [],
    exportPath: data?.exportPath,
    validation: data?.validation,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
