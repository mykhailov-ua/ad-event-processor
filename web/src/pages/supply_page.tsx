import { SupplyHub } from '@/domains/creative/supply_hub';
import { useSupplyPageWorkspace } from '@/domains/creative/use_supply_page_workspace';

export function SupplyPage() {
  return <SupplyHub {...useSupplyPageWorkspace()} />;
}
