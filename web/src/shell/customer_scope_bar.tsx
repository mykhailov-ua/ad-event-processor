import { FilterApplyButton } from '@/shell/action_buttons';
import {
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_WIDE_CLASS,
} from '@/shell/filter_panel';
import { Input } from '@/components/ui/input';
import { adminTypography } from '@/lib/admin_kit';

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
      <h2 className={adminTypography.sectionTitle}>Customer scope</h2>
      <form
        className={INLINE_FILTER_ACTION_GRID_WIDE_CLASS}
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
        <p className={adminTypography.bodyMuted}>
          Active scope:{' '}
          <span className="font-mono text-xs text-foreground">{appliedCustomerId}</span>
        </p>
      ) : (
        <p className={adminTypography.bodyMuted}>
          Set customer_id for scoped integration reads and writes.
        </p>
      )}
    </FilterPanel>
  );
}
