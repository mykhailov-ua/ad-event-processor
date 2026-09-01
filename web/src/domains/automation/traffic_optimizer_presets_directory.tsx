import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import type { TrafficOptimizerPreset } from '@/api/types';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';
import { PresetCatalogGrid } from '@/domains/automation/preset_catalog_grid';

export type TrafficOptimizerPresetsDirectoryProps = {
  items: TrafficOptimizerPreset[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function TrafficOptimizerPresetsDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
}: TrafficOptimizerPresetsDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Traffic optimizer presets">
        <AutomationNav />
        {automationPanelError(error, 'Could not load optimizer presets')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      description="Preset catalog for lander, offer, and creative weight optimization."
      title="Traffic optimizer presets"
    >
      <AutomationNav />

      {items.length === 0 ? (
        <EmptyState title="No presets" description="Traffic optimizer preset catalog is empty." />
      ) : (
        <PresetCatalogGrid items={items} />
      )}

      {error && hasSnapshot ? automationPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
