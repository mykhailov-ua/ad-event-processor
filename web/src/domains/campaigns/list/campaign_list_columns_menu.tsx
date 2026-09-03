import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_MIDDLE_COLUMNS,
  type CampaignListColumnPrefs,
  type CampaignListMiddleColumnId,
  saveCampaignListColumnPrefs,
  setMiddleColumnVisible,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  CAMPAIGN_LIST_COLUMN_CATEGORIES,
  CAMPAIGN_LIST_COLUMN_PRESET_LABELS,
  campaignListColumnPrefsFromPreset,
  defaultCampaignListPreferencesPrefs,
  type CampaignListColumnPresetId,
} from '@/domains/campaigns/list/campaign_list_preferences';

export type CampaignListColumnsMenuProps = {
  columnPrefs: CampaignListColumnPrefs;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  disabled?: boolean;
};

export function CampaignListColumnsMenu({
  columnPrefs,
  onColumnPrefsChange,
  disabled = false,
}: CampaignListColumnsMenuProps) {
  function persist(next: CampaignListColumnPrefs) {
    onColumnPrefsChange(next);
    saveCampaignListColumnPrefs(next);
  }

  function toggleColumn(columnId: CampaignListMiddleColumnId, visible: boolean) {
    persist({
      ...columnPrefs,
      hidden: setMiddleColumnVisible(columnPrefs.hidden, columnId, visible),
    });
  }

  function applyPreset(presetId: CampaignListColumnPresetId) {
    persist(campaignListColumnPrefsFromPreset(presetId));
  }

  function restoreDefault() {
    persist(defaultCampaignListPreferencesPrefs());
  }

  const hidden = new Set(columnPrefs.hidden);
  const visibleCount = CAMPAIGN_LIST_MIDDLE_COLUMNS.filter((id) => !hidden.has(id)).length;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button disabled={disabled} type="button" variant="secondary">
          Columns{visibleCount > 0 ? ` (${visibleCount})` : ''}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-72 p-0">
        <div className="flex flex-wrap gap-1 border-b border-zinc-200 p-2 dark:border-zinc-800">
          {(Object.keys(CAMPAIGN_LIST_COLUMN_PRESET_LABELS) as CampaignListColumnPresetId[]).map(
            (presetId) => (
              <button
                key={presetId}
                className="text-xs text-blue-600 dark:text-blue-400"
                type="button"
                onClick={() => applyPreset(presetId)}
              >
                {CAMPAIGN_LIST_COLUMN_PRESET_LABELS[presetId]}
              </button>
            ),
          )}
        </div>

        <div className="grid max-h-64 grid-cols-2 gap-2 overflow-y-auto p-2">
          {CAMPAIGN_LIST_COLUMN_CATEGORIES.map((category) => (
            <section key={category.id} className="flex flex-col gap-1">
              <h3 className="text-xs font-semibold text-zinc-500">{category.title}</h3>
              <ul className="grid gap-0.5">
                {category.columns.map((columnId) => {
                  const checked = !hidden.has(columnId);
                  return (
                    <li key={columnId}>
                      <label className="flex items-center gap-2 text-sm">
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(next) => toggleColumn(columnId, next === true)}
                        />
                        <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
                          {CAMPAIGN_LIST_COLUMN_LABELS[columnId]}
                        </span>
                      </label>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))}
        </div>

        <div className="border-t border-zinc-200 p-2 dark:border-zinc-800">
          <button className="text-blue-600 hover:underline dark:text-blue-400" type="button" onClick={restoreDefault}>
            Restore to default
          </button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
