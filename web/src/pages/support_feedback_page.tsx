import { useCallback, useEffect, useState } from 'react';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  fetchSupportFeedbackMeta,
  submitSupportFeedback,
  type SupportFeedbackMeta,
} from '../helpers/support_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { FeedbackForm } from '../ui/support/feedback_form.js';

export function SupportFeedbackPage() {
  const [meta, setMeta] = useState<SupportFeedbackMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchSupportFeedbackMeta(ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setMeta(result ?? null);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, []);

  const onSubmit = useCallback(
    async (body: {
      type: string;
      contact_email: string;
      message: string;
      attach_bundle?: boolean;
    }) => {
      setSubmitting(true);
      try {
        const result = await submitSupportFeedback(body);
        pushToastMessage({
          title: 'Feedback submitted',
          message: result.id ?? 'Thank you',
        });
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Submit failed',
          message: err instanceof Error ? err.message : 'Submit failed',
        });
      } finally {
        setSubmitting(false);
      }
    },
    []
  );

  return (
    <FeedbackForm
      meta={meta}
      loading={loading}
      error={error}
      submitting={submitting}
      onSubmit={(body) => {
        void onSubmit(body);
      }}
    />
  );
}
