import { RefreshCw, Settings2, BookOpen } from 'lucide-react';
import { Link } from 'react-router-dom';

import {
  DropdownMenu,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { CustomerComboboxOption } from '@/components/system/customer_combobox';
import { CampaignsListColumnSettings } from '@/domains/campaigns/campaigns_list_table';
import type { CampaignListColumnPrefs } from '@/domains/campaigns/campaign_list_columns';
import type { CampaignStatusFilter } from '@/domains/campaigns/campaigns_list_types';

const SOURCE_FILTER_OPTIONS = [{ id: '', label: 'All sources' }];

const DATE_FILTER_OPTIONS = [
  { id: 'today', label: 'Today' },
  { id: 'yesterday', label: 'Yesterday' },
  { id: 'last_7_days', label: 'Last 7 days' },
  { id: 'this_month', label: 'This month' },
];

const STATE_FILTER_OPTIONS: { id: CampaignStatusFilter; label: string }[] = [
  { id: '', label: 'All states' },
  { id: 'ACTIVE', label: 'Active' },
  { id: 'PAUSED', label: 'Paused' },
  { id: 'ARCHIVED', label: 'Archived' },
];

export type CampaignsListToolbarProps = {
  draftCustomerId: string;
  draftStatus: CampaignStatusFilter;
  customerOptions: CustomerComboboxOption[];
  columnPrefs: CampaignListColumnPrefs;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  fetching?: boolean;
  onDraftCustomerIdChange: (customerId: string) => void;
  onDraftStatusChange: (status: CampaignStatusFilter) => void;
  onRefresh: () => void;
  onCreateClick: () => void;
  onImportClick: () => void;
  onWizardClick: () => void;
};

export function CampaignsListToolbar({
  draftCustomerId,
  draftStatus,
  customerOptions,
  columnPrefs,
  onColumnPrefsChange,
  fetching = false,
  onDraftCustomerIdChange,
  onDraftStatusChange,
  onRefresh,
  onCreateClick,
  onImportClick,
  onWizardClick,
}: CampaignsListToolbarProps) {
  return (
    <div className="campaigns-list-workspace-toolbar">
      <div className="campaigns-list-workspace-toolbar-row">
        <button
          aria-label="Create campaign"
          className="campaigns-list-workspace-btn-create"
          type="button"
          onClick={onCreateClick}
        >
          Create
        </button>

        <button className="campaigns-list-workspace-btn-secondary" type="button">
          Groups
        </button>

        <Link className="campaigns-list-workspace-help-link ml-auto inline-flex items-center gap-1" to="/docs">
          <BookOpen aria-hidden className="h-3.5 w-3.5 shrink-0" />
          How to work with campaigns?
        </Link>
      </div>

      <div className="campaigns-list-workspace-toolbar-row campaigns-list-workspace-toolbar-row-filters-bar">
        <Select disabled value={SOURCE_FILTER_OPTIONS[0].id}>
          <SelectTrigger plain aria-label="Source" className="w-[8.5rem]">
            <SelectValue placeholder="All sources" />
          </SelectTrigger>
          <SelectContent plain>
            {SOURCE_FILTER_OPTIONS.map((option) => (
              <SelectItem key={option.id || 'all'} plain value={option.id || 'all'}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={draftCustomerId || '__all__'}
          onValueChange={(value) =>
            onDraftCustomerIdChange(value === '__all__' ? '' : value)
          }
        >
          <SelectTrigger plain aria-label="Customer" className="max-w-[11rem]">
            <SelectValue placeholder="All groups" />
          </SelectTrigger>
          <SelectContent plain>
            <SelectItem plain value="__all__">
              All groups
            </SelectItem>
            {customerOptions.map((customer) => (
              <SelectItem key={customer.id} plain value={customer.id}>
                {customer.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={draftStatus || '__all__'}
          onValueChange={(value) =>
            onDraftStatusChange(value === '__all__' ? '' : (value as CampaignStatusFilter))
          }
        >
          <SelectTrigger plain aria-label="Status" className="w-[8.5rem]">
            <SelectValue placeholder="All states" />
          </SelectTrigger>
          <SelectContent plain>
            {STATE_FILTER_OPTIONS.map((option) => (
              <SelectItem key={option.id || '__all__'} plain value={option.id || '__all__'}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select defaultValue="today">
          <SelectTrigger plain aria-label="Date range" className="w-[7rem]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent plain>
            {DATE_FILTER_OPTIONS.map((option) => (
              <SelectItem key={option.id} plain value={option.id}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="campaigns-list-workspace-toolbar-actions">
          <button
            aria-label="Refresh campaigns"
            className="campaigns-list-workspace-icon-btn"
            disabled={fetching}
            type="button"
            onClick={onRefresh}
          >
            <RefreshCw className={fetching ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'} />
          </button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button aria-label="Columns" className="campaigns-list-workspace-icon-btn" type="button">
                <Settings2 className="h-3.5 w-3.5" />
              </button>
            </DropdownMenuTrigger>
            <CampaignsListColumnSettings
              columnPrefs={columnPrefs}
              onColumnPrefsChange={onColumnPrefsChange}
              onImportClick={onImportClick}
              onWizardClick={onWizardClick}
            />
          </DropdownMenu>
        </div>
      </div>
    </div>
  );
}
