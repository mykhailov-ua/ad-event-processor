import { useCallback, useState } from 'react';

import { createSupportFeedback, getSupportFeedbackMeta } from '@/api/platform_api';
import { useResource } from '@/api/use_resource';
import { toError } from '@/lib/admin_error';

export type SupportFeedbackType = 'bug' | 'feature' | 'support';

export function useSupportFeedbackPageWorkspace() {
  const { data: meta, error: metaError, fetching: metaFetching } = useResource(
    (signal) => getSupportFeedbackMeta(signal),
    []
  );

  const [feedbackType, setFeedbackType] = useState<SupportFeedbackType>('bug');
  const [contactEmail, setContactEmail] = useState('');
  const [message, setMessage] = useState('');
  const [attachBundle, setAttachBundle] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<Error | undefined>();
  const [submittedId, setSubmittedId] = useState<string | undefined>();

  const onSubmit = useCallback(async () => {
    if (submitting) {
      return;
    }
    setSubmitting(true);
    setSubmitError(undefined);
    setSubmittedId(undefined);
    try {
      const response = await createSupportFeedback({
        type: feedbackType,
        contact_email: contactEmail.trim(),
        message: message.trim(),
        attach_bundle: attachBundle,
      });
      setSubmittedId(response.id);
      setMessage('');
      setAttachBundle(false);
    } catch (err) {
      setSubmitError(toError(err));
    } finally {
      setSubmitting(false);
    }
  }, [attachBundle, contactEmail, feedbackType, message, submitting]);

  return {
    meta,
    metaError,
    metaFetching,
    feedbackType,
    setFeedbackType,
    contactEmail,
    setContactEmail,
    message,
    setMessage,
    attachBundle,
    setAttachBundle,
    submitting,
    submitError,
    submittedId,
    onSubmit,
  };
}

export type SupportFeedbackPageWorkspace = ReturnType<typeof useSupportFeedbackPageWorkspace>;
