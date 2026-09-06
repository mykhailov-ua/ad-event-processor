import { OffersDirectory } from '@/domains/creative/offers_directory';
import { useOffersPageWorkspace } from '@/domains/creative/use_offers_page_workspace';

export function OffersPage() {
  return <OffersDirectory {...useOffersPageWorkspace()} />;
}
