import { FilterApplyButton } from '@/shell/action_buttons';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
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
      <DirectoryFilterForm
        className="max-w-xl grid-cols-[1fr_auto]"
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
      </DirectoryFilterForm>
      {appliedCampaignId ? (
        <p className="text-sm text-muted-foreground">
          Active scope:{' '}
          <span className="font-mono text-xs text-foreground">{appliedCampaignId}</span>
        </p>
      ) : (
        <p className="text-sm text-muted-foreground">
          Set campaign_id for scoped margin guard reads.
        </p>
      )}
    </FilterPanel>
  );
}
