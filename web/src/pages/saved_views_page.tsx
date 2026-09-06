import { SavedViewsPanel } from '@/domains/portals/saved_views_panel';
import { useSavedViewsPageWorkspace } from '@/domains/portals/use_saved_views_page_workspace';

export function SavedViewsPage() {
  return <SavedViewsPanel {...useSavedViewsPageWorkspace()} />;
}
