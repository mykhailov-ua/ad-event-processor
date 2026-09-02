import { useCallback, useState } from 'react';

import { createSupportFeedback, getSupportFeedbackMeta } from '@/api/platform_api';
import { SupportFeedbackForm } from '@/domains/platform/support_feedback_form';
import { useResource } from '@/api/use_resource';

export function SupportFeedbackPage() {
  const { data: meta, error: metaError, fetching: fetchingMeta } = useResource(
    (signal) => getSupportFeedbackMeta(signal),
    [],
  );

  const [draftType, setDraftType] = useState('bug');
  const [draftContactEmail, setDraftContactEmail] = useState('');
  const [draftMessage, setDraftMessage] = useState('');
  const [attachBundle, setAttachBundle] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<Error | undefined>(undefined);
  const [submittedId, setSubmittedId] = useState<string | undefined>(undefined);

  const onSubmit = useCallback(() => {
    const type = draftType.trim();
    const message = draftMessage.trim();
    if (!type || !message) {
      return;
    }
    setSubmitting(true);
    setSubmitError(undefined);
    setSubmittedId(undefined);
    void createSupportFeedback({
      type,
      message,
      contact_email: draftContactEmail.trim() || undefined,
      attach_bundle: attachBundle,
    })
      .then((response) => {
        setSubmittedId(response.id);
        setDraftMessage('');
      })
      .catch((err: unknown) => {
        setSubmitError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setSubmitting(false);
      });
  }, [attachBundle, draftContactEmail, draftMessage, draftType]);

  return (
    <SupportFeedbackForm
      meta={meta}
      draftType={draftType}
      draftContactEmail={draftContactEmail}
      draftMessage={draftMessage}
      attachBundle={attachBundle}
      fetchingMeta={fetchingMeta}
      submitting={submitting}
      metaError={metaError}
      submitError={submitError}
      submittedId={submittedId}
      onDraftTypeChange={setDraftType}
      onDraftContactEmailChange={setDraftContactEmail}
      onDraftMessageChange={setDraftMessage}
      onAttachBundleChange={setAttachBundle}
      onSubmit={onSubmit}
    />
  );
}
