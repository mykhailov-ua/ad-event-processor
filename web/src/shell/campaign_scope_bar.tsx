import { FilterApplyButton } from '@/shell/action_buttons';
import {
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_WIDE_CLASS,
} from '@/shell/filter_panel';
import { Input } from '@/components/ui/input';

export type CampaignScopeBarProps = {
  draftCampaignId: string;
  appliedCampaignId: string;
  onDraftCampaignIdChange: (value: string) => void;
  onApply: () => void;
};

export function CampaignScopeBar({
  draftCampaignId,
  appliedCampaignId,
  onDraftCampaignIdChange,
  onApply,
}: CampaignScopeBarProps) {
  return (
    <FilterPanel className="gap-2">
      <h2 className="text-base font-semibold">Campaign scope</h2>
      <form
        className={INLINE_FILTER_ACTION_GRID_WIDE_CLASS}
        onSubmit={(event) => {
          event.preventDefault();
          onApply();
        }}
      >
        <FilterField htmlFor="campaign-scope-id" label="Campaign ID">
          <Input
            id="campaign-scope-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </FilterField>
        <FilterApplyButton>Apply</FilterApplyButton>
      </form>
      {appliedCampaignId ? (
        <p className="text-sm text-muted-foreground">
          Active scope: <span className="text-xs text-foreground">{appliedCampaignId}</span>
        </p>
      ) : (
        <p className="text-sm text-muted-foreground">
          Set campaign_id for scoped margin guard reads.
        </p>
      )}
    </FilterPanel>
  );
}
