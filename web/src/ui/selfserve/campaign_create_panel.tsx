import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import type { SelfServeTemplate } from '../../helpers/selfserve_api.js';
import {
  createSelfServeCampaign,
  createSelfServePaymentIntent,
} from '../../helpers/selfserve_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './selfserve_shared.module.css';

export type CampaignCreatePanelProps = {
  templates: SelfServeTemplate[];
  loading: boolean;
  error: unknown;
  customerId?: string;
};

export function CampaignCreatePanel({
  templates,
  loading,
  error,
  customerId,
}: CampaignCreatePanelProps) {
  const navigate = useNavigate();
  const [templateId, setTemplateId] = useState('');
  const [name, setName] = useState('');
  const [budgetMicro, setBudgetMicro] = useState('');
  const [addPayment, setAddPayment] = useState(false);
  const [paymentMicro, setPaymentMicro] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<unknown>(null);

  if (loading && templates.length === 0) {
    return <PageSkeleton rows={5} />;
  }

  if (error && templates.length === 0) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load templates" />;
  }

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const budget = budgetMicro.trim() ? Number.parseInt(budgetMicro, 10) : undefined;
    if (budgetMicro.trim() && (!Number.isFinite(budget) || (budget ?? 0) <= 0)) {
      setSubmitError(new Error('budget_limit_micro must be a positive integer'));
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    void (async () => {
      try {
        const body: {
          template_id: string;
          name: string;
          budget_limit_micro?: number;
          customer_id?: string;
        } = {
          template_id: templateId,
          name: name.trim(),
        };
        if (budget != null) body.budget_limit_micro = budget;
        if (customerId) body.customer_id = customerId;
        const created = await createSelfServeCampaign(body);
        if (addPayment && paymentMicro.trim()) {
          const amount = Number.parseInt(paymentMicro, 10);
          if (Number.isFinite(amount) && amount > 0) {
            await createSelfServePaymentIntent({
              amount_micro: amount,
              currency: 'USD',
              customer_id: customerId,
            });
            pushToastMessage({
              title: 'Payment intent created',
              message: 'Checkout details are in the payment provider response.',
            });
          }
        }
        pushToastMessage({
          title: 'Campaign created',
          message: created.id ? `Campaign ${created.id}` : 'Campaign created',
        });
        if (created.id) {
          navigate(`/campaigns/${created.id}`);
        } else {
          navigate('/selfserve');
        }
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        setSubmitError(err);
      } finally {
        setSubmitting(false);
      }
    })();
  };

  return (
    <div data-testid="selfserve-campaign-create-panel">
      <PageChrome title="Create campaign" />
      <p className={styles.intro}>
        Templates from GET /api/v1/selfserve/templates. POST /api/v1/selfserve/campaigns requires
        Idempotency-Key.
      </p>
      {submitError ? <ErrorBlock error={submitError} fallbackTitle="Create failed" /> : null}
      <form className={styles.form} onSubmit={onSubmit}>
        <label className={styles.field}>
          <span className={styles.label}>Template</span>
          <select
            className={styles.select}
            required
            value={templateId}
            onChange={(e) => setTemplateId(e.target.value)}
          >
            <option value="">Select template</option>
            {templates.map((template) => (
              <option key={template.id} value={template.id}>
                {template.name}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Campaign name</span>
          <input
            className={styles.input}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Budget limit (micros, optional)</span>
          <input
            className={styles.input}
            inputMode="numeric"
            value={budgetMicro}
            onChange={(e) => setBudgetMicro(e.target.value)}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>
            <input
              type="checkbox"
              checked={addPayment}
              onChange={(e) => setAddPayment(e.target.checked)}
            />{' '}
            Create payment intent after campaign
          </span>
        </label>
        {addPayment ? (
          <label className={styles.field}>
            <span className={styles.label}>Payment amount (micros)</span>
            <input
              className={styles.input}
              inputMode="numeric"
              value={paymentMicro}
              onChange={(e) => setPaymentMicro(e.target.value)}
            />
          </label>
        ) : null}
        <Button type="submit" variant="primary" disabled={submitting || !templateId || !name.trim()}>
          {submitting ? 'Creating...' : 'Create campaign'}
        </Button>
      </form>
    </div>
  );
}
