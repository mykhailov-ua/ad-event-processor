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
    <div aria-label="Table view" className="flex shrink-0 flex-nowrap items-center gap-4">
      <CampaignListColumnsMenu
        columnPrefs={columnPrefs}
        disabled={disabled}
        onColumnPrefsChange={onColumnPrefsChange}
      />
      {disabled ? (
        <span className="shrink-0 whitespace-nowrap text-[13px] font-medium text-emerald-500/50">
          Reset view
        </span>
      ) : (
        <Button
          className="h-auto whitespace-nowrap border-0 bg-transparent p-0 font-medium text-emerald-500 underline underline-offset-2 shadow-none hover:bg-transparent hover:text-emerald-600"
          title="Reset columns and widths"
          type="button"
          variant="link"
          onClick={onResetWorkspaceClick}
        >
          Reset view
        </Button>
      )}
    </div>
  );
}
