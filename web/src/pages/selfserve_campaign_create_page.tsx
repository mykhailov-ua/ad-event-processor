import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { createSelfServeCampaign, fetchSelfServeTemplates } from '../helpers/selfserve_api.js';
import { ParseDecimal } from '../helpers/money.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { to } from '../lib/to.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';

export function SelfServeCampaignCreatePage() {
  const navigate = useNavigate();
  const canWrite = can(auth.getUser()?.permissions ?? [], 'campaigns:write');
  const [templates, setTemplates] = useState<
    Array<{ id: string; name: string; budget_limit: string }>
  >([]);
  const [templateId, setTemplateId] = useState('');
  const [name, setName] = useState('');
  const [budgetInput, setBudgetInput] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    void (async () => {
      const [data, err] = await to(fetchSelfServeTemplates());
      setLoading(false);
      if (err) {
        setError(err);
        return;
      }
      setTemplates(data ?? []);
      if (data?.[0]?.id) setTemplateId(data[0].id);
    })();
  }, []);

  const submit = async () => {
    if (!canWrite || busy || !templateId) return;
    const body: { template_id: string; name?: string; budget_limit_micro?: number } = {
      template_id: templateId,
    };
    if (name.trim()) body.name = name.trim();
    if (budgetInput.trim()) {
      try {
        body.budget_limit_micro = ParseDecimal(budgetInput.trim());
      } catch {
        pushToastMessage({
          title: 'Invalid budget',
          message: 'Enter a positive decimal or leave blank.',
        });
        return;
      }
    }
    setBusy(true);
    const [, err] = await to(createSelfServeCampaign(body));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Create failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Campaign created', message: 'Added from template.' });
    navigate('/selfserve');
  };

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Templates unavailable" />;
  }

  return (
    <section className="stack" data-testid="selfserve-campaign-create">
      <div className="page-header">
        <h1 className="page-header__title">Create campaign</h1>
        <p className="page-header__desc">
          Pick an approved template. Operator-only settings are not exposed here.
        </p>
      </div>
      <div className="section-card stack">
        {loading ? <p className="text-muted">Loading templates…</p> : null}
        {!loading ? (
          <>
            <label className="form-field" htmlFor="ss-template">
              Template
              <select
                id="ss-template"
                className="form-select"
                value={templateId}
                onChange={(e) => setTemplateId(e.target.value)}
              >
                {templates.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name} (budget {t.budget_limit})
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field" htmlFor="ss-camp-name">
              Display name (optional)
              <input
                id="ss-camp-name"
                className="form-input"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <label className="form-field" htmlFor="ss-camp-budget">
              Budget override USD (optional)
              <input
                id="ss-camp-budget"
                className="form-input"
                inputMode="decimal"
                value={budgetInput}
                onChange={(e) => setBudgetInput(e.target.value)}
              />
            </label>
            <Button
              label={busy ? 'Creating…' : 'Create from template'}
              variant="primary"
              loading={busy}
              disabled={!canWrite || busy || !templateId}
              onClick={() => void submit()}
            />
          </>
        ) : null}
      </div>
    </section>
  );
}
