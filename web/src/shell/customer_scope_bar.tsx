import { FilterApplyButton } from '@/shell/action_buttons';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
} from '@/shell/filter_panel';
import { Input } from '@/components/ui/input';

export type CustomerScopeBarProps = {
  draftCustomerId: string;
  appliedCustomerId: string;
  onDraftCustomerIdChange: (value: string) => void;
  onApply: () => void;
};

export function CustomerScopeBar({
  draftCustomerId,
  appliedCustomerId,
  onDraftCustomerIdChange,
  onApply,
}: CustomerScopeBarProps) {
  return (
    <FilterPanel className="gap-2">
      <h2 className="text-base font-semibold">Customer scope</h2>
      <DirectoryFilterForm
        className="max-w-xl grid-cols-[1fr_auto]"
        onSubmit={(event) => {
          event.preventDefault();
          onApply();
        }}
      >
        <FilterField htmlFor="customer-scope-id" label="Customer ID">
          <Input
            className="text-sm"
            id="customer-scope-id"
            placeholder="Customer UUID..."
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </FilterField>
        <FilterApplyButton>Apply</FilterApplyButton>
      </DirectoryFilterForm>
      {appliedCustomerId ? (
        <p className="text-sm text-muted-foreground">
          Active scope:{' '}
          <span className="font-mono text-xs text-foreground">{appliedCustomerId}</span>
        </p>
      ) : (
        <p className="text-sm text-muted-foreground">
          Set customer_id for scoped integration reads and writes.
        </p>
      )}
    </FilterPanel>
  );
}
