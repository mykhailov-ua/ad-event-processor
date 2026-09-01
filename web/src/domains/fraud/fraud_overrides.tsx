import { Link } from 'react-router-dom';

import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { ErrorBlock } from '@/components/system/error_block';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

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
    <PageChrome title="Fraud overrides">
      <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
        Back to fraud hub
      </Link>

      <div className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="override-customer-id">Customer ID</Label>
          <Input
            id="override-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <SecondaryActionButton disabled={!draftCustomerId.trim()} onClick={onApplyCustomer} type="button">
          Set customer
        </SecondaryActionButton>
      </div>

      <div className="ui-filter-panel max-w-2xl">
        <div className="grid gap-2">
          <Label htmlFor="override-campaign-id">Campaign ID (optional)</Label>
          <Input
            id="override-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="override-ip-hash">Provide IP hash or raw IP</Label>
          <Input
            id="override-ip-hash"
            placeholder="32 hex characters"
            value={draftIpHash}
            onChange={(event) => onDraftIpHashChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="override-ip">IP address</Label>
          <Input
            id="override-ip"
            placeholder="e.g. 203.0.113.42"
            value={draftIp}
            onChange={(event) => onDraftIpChange(event.target.value)}
          />
        </div>
        <PrimaryActionButton
          disabled={!customerId || !hasIpTarget}
          loading={saving}
          onClick={onSubmit}
          type="button"
        >
          Apply override
        </PrimaryActionButton>
      </div>

      {saveError ? <ErrorBlock title="Override failed" message={saveError.message} /> : null}
      {saveSuccess ? (
        <p className="text-sm text-muted-foreground">Override accepted by the API.</p>
      ) : null}
    </PageChrome>
  );
}
