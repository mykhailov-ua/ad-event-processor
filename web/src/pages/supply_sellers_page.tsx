import { SupplySellersDirectory } from '@/domains/creative/supply_sellers';
import { useSupplySellersPageWorkspace } from '@/domains/creative/use_supply_sellers_page_workspace';

export function SupplySellersPage() {
  return <SupplySellersDirectory {...useSupplySellersPageWorkspace()} />;
}
