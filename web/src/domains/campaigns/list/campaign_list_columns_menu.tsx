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
  setMiddleColumnVisible,
  visibleMiddleColumnCount,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListColumnsMenuCheckboxClass,
  campaignListColumnsMenuClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import {
  CAMPAIGN_LIST_COLUMN_CATEGORIES,
  CAMPAIGN_LIST_COLUMN_PRESET_LABELS,
  campaignListColumnPrefsFromPreset,
  defaultCampaignListPreferencesPrefs,
  type CampaignListColumnPresetId,
} from '@/domains/campaigns/list/campaign_list_preferences';
import { adminKit } from '@/lib/admin_kit';
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
          className="shrink-0 gap-1.5 whitespace-nowrap font-medium"
          disabled={disabled}
          type="button"
          variant="outline"
        >
          <Columns3 className="h-3.5 w-3.5 shrink-0" aria-hidden />
          Columns ({visibleMiddleCount}/{totalMiddleCount})
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className={cn(campaignListColumnsMenuClass, 'gap-0 p-0')}
        scrollable={false}
      >
        <div className="grid gap-1 border-b border-border px-2 py-1.5">
          <p className="m-0 text-[10px] font-semibold uppercase leading-[14px] text-muted-foreground">
            Preset views
          </p>
          <div className="flex flex-wrap items-center gap-x-2.5 gap-y-0.5">
            {PRESET_ORDER.map((presetId) => (
              <Button
                key={presetId}
                className={cn(
                  'h-auto shadow-none',
                  adminKit.controlPaddingX,
                  'text-[13px]',
                  activePreset === presetId
                    ? 'font-semibold text-foreground hover:bg-transparent'
                    : 'font-normal text-muted-foreground hover:bg-transparent hover:text-foreground'
                )}
                type="button"
                variant="ghost"
                onClick={() => applyPreset(presetId)}
              >
                {CAMPAIGN_LIST_COLUMN_PRESET_LABELS[presetId]}
              </Button>
            ))}
          </div>
        </div>

        <div className="ui-scrollbar grid max-h-72 grid-cols-4 divide-x divide-border overflow-y-auto">
          {CAMPAIGN_LIST_COLUMN_CATEGORIES.map((category) => (
            <section key={category.id} className="flex min-w-0 flex-col gap-1 px-2 py-1.5">
              <h3 className="m-0 text-[10px] font-semibold uppercase leading-[14px] text-muted-foreground">
                {category.title}
              </h3>
              <ul className="grid gap-0.5">
                {category.columns.map((columnId) => {
                  const checked = !hidden.has(columnId);
                  return (
                    <li key={columnId}>
                      <label
                        className={cn(
                          'flex cursor-pointer items-center gap-1.5 py-px',
                          adminKit.controlHeight
                        )}
                      >
                        <Checkbox
                          checked={checked}
                          className={campaignListColumnsMenuCheckboxClass}
                          onCheckedChange={(next) => toggleColumn(columnId, next === true)}
                        />
                        <span className="whitespace-nowrap text-[12px] leading-4 text-foreground/80">
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

        <div className="border-t border-border px-2 py-1.5 text-center">
          <Button
            className={cn(
              'h-auto font-medium text-primary underline-offset-2 shadow-none hover:bg-transparent hover:underline',
              adminKit.controlPaddingX,
              'text-[13px]'
            )}
            type="button"
            variant="link"
            onClick={restoreDefault}
          >
            Restore to default
          </Button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
