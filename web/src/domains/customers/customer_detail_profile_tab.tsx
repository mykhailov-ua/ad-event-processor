import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import type { Customer } from '@/api/types';
import { CustomerDetailFieldRow } from '@/domains/customers/customer_detail_field_row';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { customerDetailRowValueClass } from '@/domains/customers/customer_detail_classes';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { ErrorBlock } from '@/shell/error_block';
import { displayTimestamp } from '@/lib/display';

export type CustomerDetailProfileTabProps = {
  customer: Customer;
  draftCostCenter: string;
  onDraftCostCenterChange: (value: string) => void;
  savingCostCenter: boolean;
  costCenterSaveError: Error | undefined;
  costCenterSaveSuccess: boolean;
  canSaveCostCenter: boolean;
  onSaveCostCenter: () => void;
};

export function CustomerDetailProfileTab({
  customer,
  draftCostCenter,
  onDraftCostCenterChange,
  savingCostCenter,
  costCenterSaveError,
  costCenterSaveSuccess,
  canSaveCostCenter,
  onSaveCostCenter,
}: CustomerDetailProfileTabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Customer profile</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-6">
        <CustomerDetailPanel>
          <CustomerDetailRow label="ID" value={customer.id} />
          <CustomerDetailRow label="Name" value={customer.name} />
          <CustomerDetailRow label="Balance" value={customer.balance} />
          <CustomerDetailRow label="Currency" value={customer.currency} />
          <CustomerDetailRow label="Active campaigns" value={customer.active_campaigns} />
          <CustomerDetailRow label="Total spend" value={customer.total_spend} />
          <CustomerDetailRow
            label="Created"
            value={displayTimestamp(customer.created_at, customer.created_at_display)}
          />
          <CustomerDetailRow
            label="Updated"
            value={displayTimestamp(customer.updated_at, customer.updated_at_display)}
          />
        </CustomerDetailPanel>

        <form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            onSaveCostCenter();
          }}
        >
          <CustomerDetailPanel>
            <CustomerDetailFieldRow htmlFor="customer-cost-center" label="Cost center">
              {canSaveCostCenter ? (
                <Input
                  id="customer-cost-center"
                  value={draftCostCenter}
                  onChange={(event) => onDraftCostCenterChange(event.target.value)}
                />
              ) : (
                <p className={customerDetailRowValueClass}>{customer.cost_center ?? '-'}</p>
              )}
            </CustomerDetailFieldRow>
          </CustomerDetailPanel>
          {canSaveCostCenter ? (
            <div className={COMPACT_TOOLBAR_ROW_CLASS}>
              <PrimaryActionButton
                disabled={!canSaveCostCenter}
                loading={savingCostCenter}
                type="submit"
              >
                Save
              </PrimaryActionButton>
            </div>
          ) : null}
        </form>
        {costCenterSaveError ? (
          <ErrorBlock title="Save failed" message={costCenterSaveError.message} />
        ) : null}
        {costCenterSaveSuccess ? (
          <p className="m-0 text-sm text-muted-foreground" role="status">
            Cost center saved.
          </p>
        ) : null}
      </CardContent>
    </Card>
  );
}
