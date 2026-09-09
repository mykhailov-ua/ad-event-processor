import { FilterApplyButton } from '@/shell/action_buttons';
import {
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_WIDE_CLASS,
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
    <FilterPanel >
      <h2 >Customer scope</h2>
      <form
       
        onSubmit={(event) => {
          event.preventDefault();
          onApply();
        }}
      >
        <FilterField htmlFor="customer-scope-id" label="Customer ID">
          <Input
            id="customer-scope-id"
            placeholder="Customer UUID..."
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </FilterField>
        <FilterApplyButton>Apply</FilterApplyButton>
      </form>
      {appliedCustomerId ? (
        <p >
          Active scope: <span >{appliedCustomerId}</span>
        </p>
      ) : (
        <p >
          Set customer_id for scoped integration reads and writes.
        </p>
      )}
    </FilterPanel>
  );
}
