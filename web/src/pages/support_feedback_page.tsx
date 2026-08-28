import { useCallback, useEffect, useState } from 'react';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FormField } from '../components/form_field.js';
import { Checkbox } from '../components/checkbox.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { fetchSupportFeedbackMeta, submitSupportFeedback } from '../helpers/ops_compliance_api.js';
import type { SupportFeedbackMetaDTO } from '../types/ops_compliance.js';

const FEEDBACK_TYPES = [
  { value: 'bug', label: 'Bug report' },
  { value: 'support', label: 'Support request' },
  { value: 'feature', label: 'Feature idea' },
];

export function SupportFeedbackPage() {
  const [meta, setMeta] = useState<SupportFeedbackMetaDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [type, setType] = useState('bug');
  const [email, setEmail] = useState('');
  const [message, setMessage] = useState('');
  const [attachBundle, setAttachBundle] = useState(false);
  const [busy, setBusy] = useState(false);
  const [lastId, setLastId] = useState('');

  const loadMeta = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setMeta(await fetchSupportFeedbackMeta());
    } catch (e) {
      setError(e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadMeta();
  }, [loadMeta]);

  const submit = async () => {
    const contact = email.trim();
    const text = message.trim();
    if (!contact || !text) {
      pushToastMessage({ title: 'Missing fields', message: 'Email and message are required' });
      return;
    }
    setBusy(true);
    try {
      const res = await submitSupportFeedback({
        type,
        contact_email: contact,
        message: text,
        attach_bundle: attachBundle,
      });
      setLastId(res.id);
      setMessage('');
      pushToastMessage({
        title: 'Feedback sent',
        message: res.id ? `Ticket ${res.id}` : 'Thank you',
      });
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Submit failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <header className="page-header">
        <h1 className="h2">Support feedback</h1>
        <p className="text-muted">
          Send operator feedback to the deployment vendor. Optional bundle includes redacted logs
          only.
        </p>
      </header>

      {error ? <ErrorBlock error={error} /> : null}

      {loading ? (
        <p className="text-muted" data-testid="feedback-loading">
          Loading deployment info...
        </p>
      ) : meta ? (
        <section className="card stack text-sm text-muted" data-testid="feedback-meta">
          <div>
            Deployment: <span className="font-mono">{meta.deployment_id || '-'}</span>
          </div>
          <div>
            Binary: <span className="font-mono">{meta.binary_version || '-'}</span>
          </div>
        </section>
      ) : null}

      <section className="card stack" data-testid="feedback-form">
        <FormField label="Type" htmlFor="feedback-type">
          <select
            id="feedback-type"
            className="form-input"
            data-testid="feedback-type"
            disabled={busy}
            value={type}
            onChange={(e) => setType(e.target.value)}
          >
            {FEEDBACK_TYPES.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label="Contact email" htmlFor="feedback-email">
          <input
            id="feedback-email"
            type="email"
            className="form-input"
            data-testid="feedback-email"
            value={email}
            disabled={busy}
            onChange={(e) => setEmail(e.target.value)}
          />
        </FormField>
        <FormField label="Message" htmlFor="feedback-message">
          <textarea
            id="feedback-message"
            className="form-input"
            rows={5}
            data-testid="feedback-message"
            value={message}
            disabled={busy}
            onChange={(e) => setMessage(e.target.value)}
          />
        </FormField>
        <Checkbox
          label="Attach redacted support bundle (logs only)"
          checked={attachBundle}
          disabled={busy}
          onChange={setAttachBundle}
        />
        <Button
          label="Send feedback"
          variant="primary"
          data-testid="feedback-submit"
          disabled={busy}
          onClick={() => void submit()}
        />
        {lastId ? (
          <p className="text-muted text-sm" data-testid="feedback-last-id">
            Last submission: <span className="font-mono">{lastId}</span>
          </p>
        ) : null}
      </section>
    </>
  );
}
