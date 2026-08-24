import { useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { createCampaign, patchCampaign } from '../helpers/campaign_admin_api.js';
import { fetchSelfServeTemplates } from '../helpers/selfserve_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { ParseDecimal } from '../helpers/money.js';
import { Button } from './button.js';
import { Modal } from './modal.js';

export type CampaignWizardProps = {
  open: boolean;
  customerId: string;
  onClose: () => void;
  onCreated: (id: string) => void;
};

export function CampaignWizard({ open, customerId, onClose, onCreated }: CampaignWizardProps) {
  const [templates, setTemplates] = useState<Array<{ id: string; name: string }>>([]);
  const [templateId, setTemplateId] = useState('');
  const [name, setName] = useState('');
  const [budget, setBudget] = useState('');
  const [targetUrl, setTargetUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName('');
    setBudget('');
    setTargetUrl('');
    setBusy(false);
    setError(null);
    void (async () => {
      const [rows, err] = await to(fetchSelfServeTemplates(customerId || undefined));
      if (err || !rows?.length) {
        setTemplates([]);
        setTemplateId('');
        return;
      }
      setTemplates(rows);
      setTemplateId(rows[0]?.id ?? '');
    })();
  }, [open, customerId]);

  async function submit() {
    if (!isCustomerUuid(customerId)) {
      setError('Valid customer UUID required');
      return;
    }
    if (!templateId || !name.trim()) {
      setError('Template and name are required');
      return;
    }
    const body: Record<string, unknown> = {
      template_id: templateId,
      name: name.trim(),
    };
    const budgetTrim = budget.trim();
    if (budgetTrim) {
      try {
        body.budget_limit_micro = ParseDecimal(budgetTrim);
      } catch {
        setError('Budget must be a positive decimal');
        return;
      }
    }
    setBusy(true);
    setError(null);
    const [data, err] = await to(createCampaign(customerId, body));
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
      description="Quick create from an approved template. Use the full wizard for traffic and postback setup."
      onClose={onClose}
      testId="campaign-wizard-modal"
      actions={
        <>
          <Button label="Cancel" variant="secondary" disabled={busy} onClick={onClose} />
          <Button
            label={busy ? 'Creating...' : 'Create'}
            variant="primary"
            loading={busy}
            disabled={busy || !templateId}
            onClick={() => void submit()}
          />
        </>
      }
    >
      {error ? <p className="text-danger text-sm">{error}</p> : null}
      <label className="form-field" htmlFor="wiz-template">
        Template
        <select
          id="wiz-template"
          className="form-input"
          value={templateId}
          disabled={busy || templates.length === 0}
          onChange={(e) => setTemplateId(e.currentTarget.value)}
        >
          {templates.map((row) => (
            <option key={row.id} value={row.id}>
              {row.name}
            </option>
          ))}
        </select>
      </label>
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
        Budget override USD (optional)
        <input
          id="wiz-budget"
          className="form-input font-mono"
          inputMode="decimal"
          value={budget}
          disabled={busy}
          onInput={(e) => setBudget(e.currentTarget.value)}
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

export function useCampaignWizard() {
  const [customerId, setCustomerId] = useState<string | null>(null);
  return {
    wizardOpen: customerId != null,
    wizardCustomerId: customerId ?? '',
    openWizard: (id: string) => setCustomerId(id),
    closeWizard: () => setCustomerId(null),
  };
}
