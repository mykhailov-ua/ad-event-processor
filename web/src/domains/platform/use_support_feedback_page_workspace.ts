// L3 support feedback form: GET meta + POST create; no list fetch.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createSupportFeedback, getSupportFeedbackMeta } from '@/api/platform_api';
import { useResource } from '@/api/use_resource';

export function useSupportFeedbackPageWorkspace() {
  const {
    data: meta,
    error: metaError,
    fetching: fetchingMeta,
  } = useResource((signal) => getSupportFeedbackMeta(signal), []);

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
        toast.success('Feedback submitted');
      })
      .catch((err: unknown) => {
        const nextError = err instanceof Error ? err : new Error(String(err));
        setSubmitError(nextError);
        toast.error(nextError.message);
      })
      .finally(() => {
        setSubmitting(false);
      });
  }, [attachBundle, draftContactEmail, draftMessage, draftType]);

  return {
    meta,
    draftType,
    draftContactEmail,
    draftMessage,
    attachBundle,
    fetchingMeta,
    submitting,
    metaError,
    submitError,
    submittedId,
    onDraftTypeChange: setDraftType,
    onDraftContactEmailChange: setDraftContactEmail,
    onDraftMessageChange: setDraftMessage,
    onAttachBundleChange: setAttachBundle,
    onSubmit,
  };
}
