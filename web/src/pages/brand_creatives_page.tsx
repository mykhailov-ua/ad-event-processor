import { BrandCreativesDirectory } from '@/domains/creative/brand_creatives_directory';
import { useBrandCreativesPageWorkspace } from '@/domains/creative/use_brand_creatives_page_workspace';

export function BrandCreativesPage() {
  const workspace = useBrandCreativesPageWorkspace();

  return (
    <BrandCreativesDirectory
      {...workspace}
      onCreateCreative={() => {
        void workspace.onCreateCreative();
      }}
      onSaveCreative={() => {
        void workspace.onSaveCreative();
      }}
      onDeleteCreative={(creativeId) => {
        void workspace.onDeleteCreative(creativeId);
      }}
    />
  );
}
