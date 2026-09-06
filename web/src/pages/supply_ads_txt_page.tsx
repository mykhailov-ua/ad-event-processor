import { SupplyAdsTxtDirectory } from '@/domains/creative/supply_ads_txt';
import { useSupplyAdsTxtPageWorkspace } from '@/domains/creative/use_supply_ads_txt_page_workspace';

export function SupplyAdsTxtPage() {
  return <SupplyAdsTxtDirectory {...useSupplyAdsTxtPageWorkspace()} />;
}
