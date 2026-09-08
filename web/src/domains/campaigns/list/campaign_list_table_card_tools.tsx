import { CampaignListColumnsMenu } from '@/domains/campaigns/list/campaign_list_columns_menu';
import type { CampaignListColumnPrefs } from '@/domains/campaigns/list/campaign_list_columns';
import { Button } from '@/components/ui/button';

export type CampaignListTableCardToolsProps = {
  columnPrefs: CampaignListColumnPrefs;
  disabled?: boolean;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  onResetWorkspaceClick: () => void;
};

export function CampaignListTableCardTools({
  columnPrefs,
  disabled = false,
  onColumnPrefsChange,
  onResetWorkspaceClick,
}: CampaignListTableCardToolsProps) {
  return (
    <div aria-label="Table view" className="flex shrink-0 flex-nowrap items-center gap-2">
      <CampaignListColumnsMenu
        columnPrefs={columnPrefs}
        disabled={disabled}
        onColumnPrefsChange={onColumnPrefsChange}
      />
      <Button
        className="h-auto p-0 font-medium underline underline-offset-2 shadow-none disabled:no-underline"
        disabled={disabled}
        title="Reset columns and widths"
        type="button"
        variant="link"
        onClick={onResetWorkspaceClick}
      >
        Reset view
      </Button>
    </div>
  );
}
