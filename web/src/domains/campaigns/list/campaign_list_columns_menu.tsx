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
  defaultCampaignListRowDisplayPrefs,
  saveCampaignListRowDisplayPrefs,
  type CampaignListRowDisplayPrefs,
} from '@/domains/campaigns/list/campaign_list_row_display';
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
  rowDisplayPrefs: CampaignListRowDisplayPrefs;
  onRowDisplayPrefsChange: (prefs: CampaignListRowDisplayPrefs) => void;
  disabled?: boolean;
};

export function CampaignListColumnsMenu({
  columnPrefs,
  onColumnPrefsChange,
  rowDisplayPrefs,
  onRowDisplayPrefsChange,
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
    onRowDisplayPrefsChange(defaultCampaignListRowDisplayPrefs());
    saveCampaignListRowDisplayPrefs(defaultCampaignListRowDisplayPrefs());
  }

  function toggleHighlightActiveRows(enabled: boolean) {
    const next = { ...rowDisplayPrefs, highlightActiveRows: enabled };
    onRowDisplayPrefsChange(next);
    saveCampaignListRowDisplayPrefs(next);
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
      <DropdownMenuContent align="end" className="admin-columns-menu p-0">
        <div className="admin-columns-menu__presets">
          {(Object.keys(CAMPAIGN_LIST_COLUMN_PRESET_LABELS) as CampaignListColumnPresetId[]).map(
            (presetId) => (
              <button
                key={presetId}
                className="admin-columns-menu__preset"
                type="button"
                onClick={() => applyPreset(presetId)}
              >
                {CAMPAIGN_LIST_COLUMN_PRESET_LABELS[presetId]}
              </button>
            ),
          )}
        </div>

        <div className="admin-columns-menu__grid">
          {CAMPAIGN_LIST_COLUMN_CATEGORIES.map((category) => (
            <section key={category.id} className="admin-columns-menu__col">
              <h3 className="admin-columns-menu__title">{category.title}</h3>
              <ul className="admin-columns-menu__list">
                {category.columns.map((columnId) => {
                  const checked = !hidden.has(columnId);
                  return (
                    <li key={columnId}>
                      <label className="admin-columns-menu__item">
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(next) => toggleColumn(columnId, next === true)}
                        />
                        <span className="admin-columns-menu__label">
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

        <div className="admin-columns-menu__footer">
          <label className="admin-columns-menu__item">
            <Checkbox
              checked={rowDisplayPrefs.highlightActiveRows}
              onCheckedChange={(next) => toggleHighlightActiveRows(next === true)}
            />
            <span className="admin-columns-menu__label">Highlight active rows</span>
          </label>
          <button className="admin-text-link" type="button" onClick={restoreDefault}>
            Restore to default
          </button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
