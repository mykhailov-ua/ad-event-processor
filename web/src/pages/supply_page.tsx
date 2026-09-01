import {
  getSupplyExportPath,
  getSupplyValidation,
  listSupplyAdsTxt,
  listSupplySellers,
} from '@/api/supply_api';
import { SupplyHub } from '@/domains/creative/supply_hub';
import { useResource } from '@/hooks/use_resource';

type SupplySnapshot = {
  sellers: Awaited<ReturnType<typeof listSupplySellers>>;
  adsTxt: Awaited<ReturnType<typeof listSupplyAdsTxt>>;
  exportPath: Awaited<ReturnType<typeof getSupplyExportPath>>;
  validation: Awaited<ReturnType<typeof getSupplyValidation>>;
};

export function SupplyPage() {
  const { data, error, fetching } = useResource<SupplySnapshot>(
    async (signal) => {
      const [sellers, adsTxt, exportPath, validation] = await Promise.all([
        listSupplySellers(signal),
        listSupplyAdsTxt(signal),
        getSupplyExportPath(signal),
        getSupplyValidation(signal),
      ]);
      return { sellers, adsTxt, exportPath, validation };
    },
    [],
  );

  return (
    <SupplyHub
      sellers={data?.sellers ?? []}
      adsTxt={data?.adsTxt ?? []}
      exportPath={data?.exportPath}
      validation={data?.validation}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
