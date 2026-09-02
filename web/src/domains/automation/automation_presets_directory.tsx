import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import type { AutomationPreset } from '@/api/types';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';
import { PresetCatalogGrid } from '@/domains/automation/preset_catalog_grid';

export type AutomationPresetsDirectoryProps = {
  items: AutomationPreset[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function AutomationPresetsDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
}: AutomationPresetsDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Automation presets">
        <AutomationNav />
        {automationPanelError(error, 'Could not load automation presets')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      description="Bundled templates for pacing, budget, and delivery guardrails."
      title="Automation presets"
    >
      <AutomationNav />

      {items.length === 0 ? (
        <EmptyState title="No presets" description="Automation preset catalog is empty." />
      ) : (
        <PresetCatalogGrid items={items} />
      )}

      {error && hasSnapshot ? automationPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
