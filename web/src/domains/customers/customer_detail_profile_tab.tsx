import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import type { Customer } from '@/api/types';
import { CustomerDetailFieldRow } from '@/domains/customers/customer_detail_field_row';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { customerDetailRowValueClass } from '@/domains/customers/customer_detail_classes';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { panelError } from '@/shell/panel_error';
import { displayTimestamp } from '@/lib/display';

export type CustomerDetailProfileTabProps = {
  customer: Customer;
  draftName: string;
  onDraftNameChange: (value: string) => void;
  draftCostCenter: string;
  onDraftCostCenterChange: (value: string) => void;
  savingProfile: boolean;
  profileSaveError: Error | undefined;
  profileSaveSuccess: boolean;
  canSaveProfile: boolean;
  onSaveProfile: () => void;
};

export function CustomerDetailProfileTab({
  customer,
  draftName,
  onDraftNameChange,
  draftCostCenter,
  onDraftCostCenterChange,
  savingProfile,
  profileSaveError,
  profileSaveSuccess,
  canSaveProfile,
  onSaveProfile,
}: CustomerDetailProfileTabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle >Customer profile</CardTitle>
      </CardHeader>
      <CardContent >
        <CustomerDetailPanel>
          <CustomerDetailRow label="ID" value={customer.id} />
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
         
          onSubmit={(event) => {
            event.preventDefault();
            onSaveProfile();
          }}
        >
          <CustomerDetailPanel>
            <CustomerDetailFieldRow htmlFor="customer-name" label="Name">
              {canSaveProfile ? (
                <Input
                  id="customer-name"
                  value={draftName}
                  onChange={(event) => onDraftNameChange(event.target.value)}
                />
              ) : (
                <p >{customer.name}</p>
              )}
            </CustomerDetailFieldRow>
            <CustomerDetailFieldRow htmlFor="customer-cost-center" label="Cost center">
              {canSaveProfile ? (
                <Input
                  id="customer-cost-center"
                  value={draftCostCenter}
                  onChange={(event) => onDraftCostCenterChange(event.target.value)}
                />
              ) : (
                <p >{customer.cost_center ?? '-'}</p>
              )}
            </CustomerDetailFieldRow>
          </CustomerDetailPanel>
          {canSaveProfile ? (
            <div >
              <PrimaryActionButton loading={savingProfile} type="submit">
                Save profile
              </PrimaryActionButton>
            </div>
          ) : null}
        </form>
        {profileSaveError ? panelError(profileSaveError, 'Save failed') : null}
        {profileSaveSuccess ? (
          <p  role="status">
            Profile saved.
          </p>
        ) : null}
      </CardContent>
    </Card>
  );
}
