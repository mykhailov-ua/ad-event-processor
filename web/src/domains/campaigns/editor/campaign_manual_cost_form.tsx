import { useState } from 'react';
import { toast } from 'sonner';

import { putManualCampaignCost } from '@/api/campaigns_api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { FilterField } from '@/shell/filter_panel';
import { toError } from '@/lib/admin_error';
import { ErrorBlock } from '@/shell/error_block';

export type CampaignManualCostFormProps = {
  campaignId: string;
  onSaved?: () => void;
};

export function CampaignManualCostForm({ campaignId, onSaved }: CampaignManualCostFormProps) {
  const [costDate, setCostDate] = useState('');
  const [amountMicro, setAmountMicro] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<Error | undefined>();

  async function onSave() {
    setSaving(true);
    setError(undefined);
    try {
      const parsed = Number(amountMicro);
      if (!costDate || !Number.isFinite(parsed) || parsed < 0) {
        throw new Error('Enter cost date and non-negative amount in micro-units.');
      }
      await putManualCampaignCost(campaignId, {
        cost_date: costDate,
        amount_micro: Math.trunc(parsed),
        currency: 'USD',
      });
      toast.success('Manual cost saved');
      onSaved?.();
    } catch (err: unknown) {
      setError(toError(err));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="grid gap-3 md:grid-cols-3">
      <FilterField htmlFor="manual-cost-date" label="Cost date">
        <Input
          id="manual-cost-date"
          type="date"
          value={costDate}
          onChange={(e) => setCostDate(e.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="manual-cost-micro" label="Amount micro">
        <Input
          id="manual-cost-micro"
          inputMode="numeric"
          value={amountMicro}
          onChange={(e) => setAmountMicro(e.target.value)}
        />
      </FilterField>
      <div className="flex items-end">
        <Button disabled={saving} onClick={onSave} type="button">
          {saving ? 'Saving...' : 'Save manual cost'}
        </Button>
      </div>
      {error ? <ErrorBlock error={error} title="Manual cost save failed" /> : null}
    </div>
  );
}
