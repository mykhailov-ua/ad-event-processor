import { useMemo } from 'react';
import { Columns3 } from 'lucide-react';

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
  visibleMiddleColumnCount,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  CAMPAIGN_LIST_COLUMN_CATEGORIES,
  CAMPAIGN_LIST_COLUMN_PRESET_LABELS,
  campaignListColumnPrefsFromPreset,
  defaultCampaignListPreferencesPrefs,
  type CampaignListColumnPresetId,
} from '@/domains/campaigns/list/campaign_list_preferences';
import { cn } from '@/lib/utils';

export type CampaignListColumnsMenuProps = {
  columnPrefs: CampaignListColumnPrefs;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  disabled?: boolean;
};

const PRESET_ORDER: CampaignListColumnPresetId[] = ['full', 'traffic', 'finance', 'minimal'];

function hiddenSignature(hidden: CampaignListMiddleColumnId[]): string {
  return [...hidden].sort().join(',');
}

function detectActivePreset(prefs: CampaignListColumnPrefs): CampaignListColumnPresetId | null {
  const signature = hiddenSignature(prefs.hidden);
  for (const presetId of PRESET_ORDER) {
    if (hiddenSignature(campaignListColumnPrefsFromPreset(presetId).hidden) === signature) {
      return presetId;
    }
  }
  return null;
}

export function CampaignListColumnsMenu({
  columnPrefs,
  onColumnPrefsChange,
  disabled = false,
}: CampaignListColumnsMenuProps) {
  const activePreset = useMemo(() => detectActivePreset(columnPrefs), [columnPrefs]);

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
  const visibleMiddleCount = visibleMiddleColumnCount(columnPrefs);
  const totalMiddleCount = CAMPAIGN_LIST_MIDDLE_COLUMNS.length;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          className="admin-campaigns-toolbar__outline-btn shrink-0 whitespace-nowrap px-2 font-medium"
          disabled={disabled}
          type="button"
          variant="outline"
        >
          <Columns3 className="mr-1 h-3.5 w-3.5 shrink-0" aria-hidden />
          Columns ({visibleMiddleCount}/{totalMiddleCount})
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="campaign-columns-menu p-0">
        <div className="border-b border-border px-3 py-2.5">
          <p className="mb-1.5 text-[10px] font-semibold uppercase leading-[14px] text-muted-foreground">
            Preset views
          </p>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            {PRESET_ORDER.map((presetId) => (
              <button
                key={presetId}
                className={cn(
                  'text-[13px] leading-[18px] transition-colors',
                  activePreset === presetId
                    ? 'font-semibold text-foreground'
                    : 'font-normal text-muted-foreground hover:text-foreground',
                )}
                type="button"
                onClick={() => applyPreset(presetId)}
              >
                {CAMPAIGN_LIST_COLUMN_PRESET_LABELS[presetId]}
              </button>
            ))}
          </div>
        </div>

        <div className="ui-scrollbar grid max-h-80 grid-cols-2 divide-x divide-border overflow-y-auto">
          {CAMPAIGN_LIST_COLUMN_CATEGORIES.map((category) => (
            <section key={category.id} className="px-3 py-2.5">
              <h3 className="mb-2 text-[10px] font-semibold uppercase leading-[14px] text-muted-foreground">
                {category.title}
              </h3>
              <ul className="grid gap-1.5">
                {category.columns.map((columnId) => {
                  const checked = !hidden.has(columnId);
                  return (
                    <li key={columnId}>
                      <label className="flex min-h-8 cursor-pointer items-center gap-2">
                        <Checkbox
                          checked={checked}
                          className="campaign-columns-menu__checkbox"
                          onCheckedChange={(next) => toggleColumn(columnId, next === true)}
                        />
                        <span className="truncate text-[13px] leading-[18px] text-foreground/80">
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

        <div className="border-t border-border px-3 py-2.5 text-center">
          <button
            className="text-[13px] font-medium leading-[18px] text-primary underline-offset-2 hover:underline"
            type="button"
            onClick={restoreDefault}
          >
            Restore to default
          </button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
