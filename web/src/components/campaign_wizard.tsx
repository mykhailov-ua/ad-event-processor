import { useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { createCampaign, patchCampaign } from '../helpers/campaign_admin_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { Button } from './button.js';
import { Modal } from './modal.js';

export type CampaignWizardProps = {
  open: boolean;
  customerId: string;
  onClose: () => void;
  onCreated: (id: string) => void;
};

/**
 * Campaign create wizard (budget reserved from customer balance on create).
 */
export function CampaignWizard({
  open,
  customerId,
  onClose,
  onCreated,
}: CampaignWizardProps) {
  const [name, setName] = useState('');
  const [budget, setBudget] = useState('100.00');
  const [pacing, setPacing] = useState('ASAP');
  const [timezone, setTimezone] = useState('UTC');
  const [countries, setCountries] = useState('US');
  const [targetUrl, setTargetUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName('');
    setBudget('100.00');
    setPacing('ASAP');
    setTimezone('UTC');
    setCountries('US');
    setTargetUrl('');
    setBusy(false);
    setError(null);
  }, [open]);

  async function submit() {
    if (!isCustomerUuid(customerId)) {
      setError('Valid customer UUID required');
      return;
    }
    const budgetNum = Number.parseFloat(budget);
    if (!name.trim() || !Number.isFinite(budgetNum) || budgetNum <= 0) {
      setError('Name and positive budget are required');
      return;
    }
    setBusy(true);
    setError(null);
    const countryList = countries.split(',').map((c) => c.trim().toUpperCase()).filter(Boolean);
    const [data, err] = await to(createCampaign(customerId, {
      name: name.trim(),
      budget_limit: budgetNum,
      pacing_mode: pacing,
      timezone: timezone.trim() || 'UTC',
      target_countries: countryList,
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        setBusy(false);
        return;
      }
      setError(mapServiceError(err).message);
      setBusy(false);
      return;
    }
    pushToastMessage({ title: 'Campaign created', message: data?.id ?? 'OK' });
    const landing = targetUrl.trim();
    if (data?.id && landing) {
      const [, patchErr] = await to(patchCampaign(data.id, { target_url: landing }));
      if (patchErr) {
        pushToastMessage({
          title: 'Landing URL',
          message: patchErr.message ?? 'Campaign created; landing URL not saved',
        });
      }
    }
    onClose();
    if (data?.id) onCreated(data.id);
  }

  return (
    <Modal
      open={open}
      title="Create campaign"
      description="Budget is reserved from customer balance on create."
      onClose={onClose}
      testId="campaign-wizard-modal"
      actions={(
        <>
          <Button label="Cancel" variant="secondary" disabled={busy} onClick={onClose} />
          <Button
            label={busy ? 'Creating…' : 'Create'}
            variant="primary"
            loading={busy}
            disabled={busy}
            onClick={() => void submit()}
          />
        </>
      )}
    >
      {error ? <p className="text-danger text-sm">{error}</p> : null}
      <label className="form-field" htmlFor="wiz-name">
        Name
        <input
          id="wiz-name"
          className="form-input"
          value={name}
          disabled={busy}
          onInput={(e) => setName(e.currentTarget.value)}
        />
      </label>
      <label className="form-field" htmlFor="wiz-budget">
        Budget (USD)
        <input
          id="wiz-budget"
          className="form-input font-mono"
          inputMode="decimal"
          value={budget}
          disabled={busy}
          onInput={(e) => setBudget(e.currentTarget.value)}
        />
      </label>
      <label className="form-field" htmlFor="wiz-pacing">
        Pacing
        <select
          id="wiz-pacing"
          className="form-input"
          value={pacing}
          disabled={busy}
          onChange={(e) => setPacing(e.currentTarget.value)}
        >
          <option value="ASAP">ASAP</option>
          <option value="EVEN">Even</option>
        </select>
      </label>
      <label className="form-field" htmlFor="wiz-tz">
        Timezone
        <input
          id="wiz-tz"
          className="form-input"
          value={timezone}
          disabled={busy}
          onInput={(e) => setTimezone(e.currentTarget.value)}
        />
      </label>
      <label className="form-field" htmlFor="wiz-countries">
        Target countries (comma-separated ISO codes)
        <input
          id="wiz-countries"
          className="form-input"
          value={countries}
          disabled={busy}
          onInput={(e) => setCountries(e.currentTarget.value)}
        />
      </label>
      <label className="form-field" htmlFor="wiz-target-url">
        Landing URL (optional)
        <input
          id="wiz-target-url"
          className="form-input"
          type="url"
          placeholder="https://example.com/offer"
          value={targetUrl}
          disabled={busy}
          onInput={(e) => setTargetUrl(e.currentTarget.value)}
        />
      </label>
    </Modal>
  );
}

/**
 * Hook for opening the campaign create wizard from list pages.
 */
export function useCampaignWizard() {
  const [customerId, setCustomerId] = useState<string | null>(null);
  return {
    wizardOpen: customerId != null,
    wizardCustomerId: customerId ?? '',
    openWizard: (id: string) => setCustomerId(id),
    closeWizard: () => setCustomerId(null),
  };
}
