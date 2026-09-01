import { FilterApplyButton } from '@/components/system/action_buttons';
import { CustomerCombobox, type CustomerComboboxOption } from '@/components/system/customer_combobox';
import { FilterField, FilterPanel } from '@/components/system/filter_panel';
import { Button } from '@/components/ui/button';
import { DateRangePicker } from '@/components/ui/date_range_picker';
import { DashboardPreferencesDialog } from '@/domains/dashboards/dashboard_preferences_dialog';
import type { BuyerDashboardPreferences } from '@/domains/dashboards/dashboard_preferences';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Settings2 } from 'lucide-react';
import { useState } from 'react';

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

  return (
    <FilterPanel className="gap-3 p-3 md:p-4">
      <p className="text-xs text-muted-foreground">
        Period and campaign apply immediately. Apply confirms customer change.
      </p>
      <div className="grid gap-3 md:grid-cols-[minmax(0,1.2fr)_minmax(0,1.2fr)_minmax(0,1.3fr)_auto_auto] md:items-end">
        <FilterField htmlFor="buyer-dashboard-customer" label="Customer">
          <CustomerCombobox
            id="buyer-dashboard-customer"
            disabled={fetching}
            options={customerOptions}
            value={draftCustomerId}
            onValueChange={onDraftCustomerIdChange}
          />
        </FilterField>
        <FilterField htmlFor="buyer-dashboard-campaign" label="Campaign">
          <Select
            value={draftCampaignId || '__all__'}
            onValueChange={(value) =>
              onDraftCampaignIdChange(value === '__all__' ? '' : value)
            }
          >
            <SelectTrigger id="buyer-dashboard-campaign" className="h-9">
              <SelectValue placeholder="All campaigns" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">All campaigns</SelectItem>
              {campaignOptions.map((campaign) => (
                <SelectItem key={campaign.id} value={campaign.id}>
                  {campaign.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>
        <DateRangePicker
          id="buyer-dashboard-range"
          className="min-w-0"
          label="Period"
          disabled={fetching || !draftCustomerId.trim()}
          from={draftFrom}
          to={draftTo}
          onChange={onDraftRangeChange}
        />
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="h-9 w-9 shrink-0 self-end"
          aria-label="Dashboard preferences"
          onClick={() => setPreferencesOpen(true)}
        >
          <Settings2 className="h-4 w-4" />
        </Button>
        {showApply ? (
          <FilterApplyButton
            className="self-end"
            disabled={fetching || !draftCustomerId.trim()}
            onClick={onApply}
            type="button"
          >
            Apply
          </FilterApplyButton>
        ) : null}
      </div>
      <DashboardPreferencesDialog
        open={preferencesOpen}
        preferences={preferences}
        onOpenChange={setPreferencesOpen}
        onApply={onPreferencesApply}
      />
    </FilterPanel>
  );
}
