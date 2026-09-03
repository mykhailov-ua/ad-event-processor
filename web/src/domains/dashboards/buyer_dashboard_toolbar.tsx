import { useMemo, useState } from 'react';
import { Settings2 } from 'lucide-react';

import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { Button } from '@/components/ui/button';
import { DateRangePicker } from '@/components/ui/date_range_picker';
import { CampaignsListFilterSelect } from '@/domains/campaigns/list/campaigns_list_filter_select';
import { DashboardPreferencesDialog } from '@/domains/dashboards/dashboard_preferences_dialog';
import type { BuyerDashboardPreferences } from '@/domains/dashboards/dashboard_preferences';

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
  fetching: boolean;
  showApply: boolean;
  preferences: BuyerDashboardPreferences;
  onPreferencesApply: (preferences: BuyerDashboardPreferences) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftRangeChange: (from: string, to: string) => void;
  onApply: () => void;
};

export function BuyerDashboardToolbar({
  draftCustomerId,
  draftCampaignId,
  draftFrom,
  draftTo,
  customerOptions,
  campaignOptions,
  fetching,
  showApply,
  preferences,
  onPreferencesApply,
  onDraftCustomerIdChange,
  onDraftCampaignIdChange,
  onDraftRangeChange,
  onApply,
}: BuyerDashboardToolbarProps) {
  const [preferencesOpen, setPreferencesOpen] = useState(false);

  const customerSelectOptions = useMemo(
    () => [
      { value: ALL_OPTION_VALUE, label: 'All customers' },
      ...customerOptions.map((customer) => ({
        value: customer.id,
        label: customer.name,
      })),
    ],
    [customerOptions],
  );

  const campaignSelectOptions = useMemo(
    () => [
      { value: ALL_OPTION_VALUE, label: 'All campaigns' },
      ...campaignOptions.map((campaign) => ({
        value: campaign.id,
        label: campaign.name,
      })),
    ],
    [campaignOptions],
  );

  return (
    <div className="flex flex-col gap-3 flex flex-col gap-2">
      <p className="flex flex-wrap items-center gap-2 text-sm text-zinc-500 dark:text-zinc-400">
        Period and campaign apply immediately. Apply confirms customer change.
      </p>
      <div
        aria-label="Dashboard filters"
        className="flex flex-wrap items-center gap-2 flex flex-wrap items-center gap-2"
        role="search"
      >
        <CampaignsListFilterSelect
          aria-label="Customer"
          disabled={fetching}
          options={customerSelectOptions}
          value={draftCustomerId || ALL_OPTION_VALUE}
          onValueChange={(value) =>
            onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
          }
        />
        <CampaignsListFilterSelect
          aria-label="Campaign"
          disabled={fetching || !draftCustomerId.trim()}
          options={campaignSelectOptions}
          value={draftCampaignId || ALL_OPTION_VALUE}
          onValueChange={(value) =>
            onDraftCampaignIdChange(value === ALL_OPTION_VALUE ? '' : value)
          }
        />
        <DateRangePicker
          className="flex flex-col gap-1"
          id="buyer-dashboard-range"
          disabled={fetching || !draftCustomerId.trim()}
          from={draftFrom}
          label="Period"
          to={draftTo}
          variant="admin"
          onChange={onDraftRangeChange}
        />
        <Button
          aria-label="Dashboard preferences"
          size="icon"
          type="button"
          variant="secondary"
          onClick={() => setPreferencesOpen(true)}
        >
          <Settings2 aria-hidden className="h-4 w-4" />
        </Button>
        {showApply ? (
          <Button disabled={fetching || !draftCustomerId.trim()} type="button" onClick={onApply}>
            Apply
          </Button>
        ) : null}
      </div>
      <DashboardPreferencesDialog
        open={preferencesOpen}
        preferences={preferences}
        onOpenChange={setPreferencesOpen}
        onApply={onPreferencesApply}
      />
    </div>
  );
}
