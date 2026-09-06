import { useMemo, useState } from 'react';
import { Settings2 } from 'lucide-react';

import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { Button } from '@/components/ui/button';
import { ToolbarDateRangePicker } from '@/shell/toolbar_date_range_picker';
import { AdminSelect } from '@/shell/admin_select';
import {
  dashboardFilterFieldClass,
  dashboardFilterLabelClass,
} from '@/domains/dashboards/dashboard_classes';
import { DashboardPreferencesDialog } from '@/domains/dashboards/dashboard_preferences_dialog';
import type { BuyerDashboardPreferences } from '@/domains/dashboards/dashboard_preferences';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';

const ALL_OPTION_VALUE = '__all__';

export type BuyerDashboardCampaignOption = {
  id: string;
  name: string;
};

export type BuyerDashboardToolbarProps = {
  draftCustomerId: string;
  draftCampaignId: string;
  draftFrom: string;
  draftTo: string;
  customerOptions: CustomerComboboxOption[];
  campaignOptions: BuyerDashboardCampaignOption[];
  preferences: BuyerDashboardPreferences;
  onPreferencesApply: (preferences: BuyerDashboardPreferences) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftRangeChange: (from: string, to: string) => void;
};

export function BuyerDashboardToolbar({
  draftCustomerId,
  draftCampaignId,
  draftFrom,
  draftTo,
  customerOptions,
  campaignOptions,
  preferences,
  onPreferencesApply,
  onDraftCustomerIdChange,
  onDraftCampaignIdChange,
  onDraftRangeChange,
}: BuyerDashboardToolbarProps) {
  const [preferencesOpen, setPreferencesOpen] = useState(false);
  const customerSelected = draftCustomerId.trim().length > 0;

  const customerSelectOptions = useMemo(
    () => [
      { value: ALL_OPTION_VALUE, label: 'All customers' },
      ...customerOptions.map((customer) => ({
        value: customer.id,
        label: customer.name,
      })),
    ],
    [customerOptions]
  );

  const campaignSelectOptions = useMemo(
    () => [
      { value: ALL_OPTION_VALUE, label: 'All campaigns' },
      ...campaignOptions.map((campaign) => ({
        value: campaign.id,
        label: campaign.name,
      })),
    ],
    [campaignOptions]
  );

  return (
    <>
      <FilterPanel aria-label="Dashboard filters" className="bg-transparent p-0" role="search">
        <DirectoryFilterForm layout="campaigns">
          <FilterField
            className={dashboardFilterFieldClass}
            label="Customer"
            labelClassName={dashboardFilterLabelClass}
          >
            <AdminSelect
              aria-label="Customer"
              options={customerSelectOptions}
              value={draftCustomerId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField
            className={dashboardFilterFieldClass}
            label="Campaign"
            labelClassName={dashboardFilterLabelClass}
          >
            <AdminSelect
              aria-label="Campaign"
              disabled={!customerSelected}
              options={campaignSelectOptions}
              title={customerSelected ? 'Filter by campaign' : 'Select a customer first'}
              value={draftCampaignId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCampaignIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField
            className={dashboardFilterFieldClass}
            htmlFor="buyer-dashboard-range"
            label="Period"
            labelClassName={dashboardFilterLabelClass}
          >
            <ToolbarDateRangePicker
              disabled={!customerSelected}
              from={draftFrom}
              id="buyer-dashboard-range"
              label="Period"
              labelClassName="sr-only"
              to={draftTo}
              onChange={onDraftRangeChange}
            />
          </FilterField>

          <div className="flex items-end">
            <Button
              aria-label="Dashboard preferences"
              className="size-7 p-0"
              type="button"
              variant="secondary"
              onClick={() => setPreferencesOpen(true)}
            >
              <Settings2 aria-hidden className="h-4 w-4" />
            </Button>
          </div>
        </DirectoryFilterForm>
      </FilterPanel>

      <DashboardPreferencesDialog
        open={preferencesOpen}
        preferences={preferences}
        onApply={onPreferencesApply}
        onOpenChange={setPreferencesOpen}
      />
    </>
  );
}
