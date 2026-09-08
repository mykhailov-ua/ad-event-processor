import { Link } from 'react-router-dom';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { ErrorBlock } from '@/shell/error_block';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import { Input } from '@/components/ui/input';

export type FraudOverridesProps = {
  customerId: string;
  draftCustomerId: string;
  draftCampaignId: string;
  draftIpHash: string;
  draftIp: string;
  saving: boolean;
  saveError: Error | undefined;
  saveSuccess: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftIpHashChange: (value: string) => void;
  onDraftIpChange: (value: string) => void;
  onApplyCustomer: () => void;
  onSubmit: () => void;
};

export function FraudOverrides({
  customerId,
  draftCustomerId,
  draftCampaignId,
  draftIpHash,
  draftIp,
  saving,
  saveError,
  saveSuccess,
  onDraftCustomerIdChange,
  onDraftCampaignIdChange,
  onDraftIpHashChange,
  onDraftIpChange,
  onApplyCustomer,
  onSubmit,
}: FraudOverridesProps) {
  const hasIpTarget = Boolean(draftIpHash.trim() || draftIp.trim());
  return (
    <PageChrome
      title="Fraud overrides"
      controlPanel={
        <div className="grid gap-3">
          <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
            Back to fraud hub
          </Link>
          <FilterPanel>
            <DirectoryFilterForm
              className={INLINE_FILTER_ACTION_GRID_CLASS}
              onSubmit={(event) => event.preventDefault()}
            >
              <FilterField htmlFor="override-customer-id" label="Customer ID">
                <Input
                  id="override-customer-id"
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </FilterField>
              <SecondaryActionButton
                disabled={!draftCustomerId.trim()}
                onClick={onApplyCustomer}
                type="button"
              >
                Set customer
              </SecondaryActionButton>
            </DirectoryFilterForm>
          </FilterPanel>
          <FilterPanel className="w-full max-w-2xl gap-4">
            <FilterField htmlFor="override-campaign-id" label="Campaign ID (optional)">
              <Input
                id="override-campaign-id"
                value={draftCampaignId}
                onChange={(event) => onDraftCampaignIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="override-ip-hash" label="Provide IP hash or raw IP">
              <Input
                id="override-ip-hash"
                placeholder="32 hex characters"
                value={draftIpHash}
                onChange={(event) => onDraftIpHashChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="override-ip" label="IP address">
              <Input
                id="override-ip"
                placeholder="e.g. 203.0.113.42"
                value={draftIp}
                onChange={(event) => onDraftIpChange(event.target.value)}
              />
            </FilterField>
            <PrimaryActionButton
              disabled={!customerId || !hasIpTarget}
              loading={saving}
              onClick={onSubmit}
              type="button"
            >
              Apply override
            </PrimaryActionButton>
          </FilterPanel>
        </div>
      }
    >
      {saveError ? <ErrorBlock title="Override failed" message={saveError.message} /> : null}
      {saveSuccess ? (
        <p className="text-sm text-muted-foreground">Override applied successfully.</p>
      ) : null}
    </PageChrome>
  );
}
