import { BrandsDirectory } from '@/domains/creative/brands_directory';
import { useBrandsPageWorkspace } from '@/domains/creative/use_brands_page_workspace';

export function BrandsPage() {
  return <BrandsDirectory {...useBrandsPageWorkspace()} />;
}
