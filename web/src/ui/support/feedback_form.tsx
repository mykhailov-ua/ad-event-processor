import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import type { SupportFeedbackMeta } from '../../helpers/support_api.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from '../settings/settings_shared.module.css';

export type FeedbackFormProps = {
  meta: SupportFeedbackMeta | null;
  loading: boolean;
  error: unknown;
  submitting: boolean;
  onSubmit: (body: {
    type: string;
    contact_email: string;
    message: string;
    attach_bundle?: boolean;
  }) => void;
};

export function FeedbackForm({
  meta,
  loading,
  error,
  submitting,
  onSubmit,
}: FeedbackFormProps) {
  const [type, setType] = useState('bug');
  const [contactEmail, setContactEmail] = useState('');
  const [message, setMessage] = useState('');
  const [attachBundle, setAttachBundle] = useState(false);

  if (error && !meta) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load feedback form" />;
  }

  const onFormSubmit = (event: FormEvent) => {
    event.preventDefault();
    const trimmedMessage = message.trim();
    const trimmedEmail = contactEmail.trim();
    if (!trimmedMessage || !trimmedEmail) return;
    onSubmit({
      type,
      contact_email: trimmedEmail,
      message: trimmedMessage,
      attach_bundle: attachBundle,
    });
    setMessage('');
  };

  return (
    <div className={styles.root} data-testid="support-feedback-page">
      <PageChrome
        title="Support feedback"
        badge={
          <Link to="/settings" className={styles.bannerLink}>
            Settings
          </Link>
        }
      />
      {loading && !meta ? (
        <PageSkeleton rows={4} />
      ) : (
        <>
          {meta ? (
            <p className={styles.hint}>
              Deployment {meta.deployment_id ?? '-'} Â. build {meta.binary_version ?? '-'}
            </p>
          ) : null}
          <form className={styles.formStack} onSubmit={onFormSubmit}>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Type</span>
              <select
                className={styles.select}
                value={type}
                onChange={(e) => setType(e.target.value)}
              >
                <option value="bug">Bug</option>
                <option value="feature">Feature</option>
                <option value="support">Support</option>
              </select>
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Contact email</span>
              <input
                className={styles.textInput}
                type="email"
                value={contactEmail}
                onChange={(e) => setContactEmail(e.target.value)}
                required
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Message</span>
              <textarea
                className={styles.textarea}
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                required
              />
            </label>
            <label className={styles.checkboxRow}>
              <input
                type="checkbox"
                checked={attachBundle}
                onChange={(e) => setAttachBundle(e.target.checked)}
              />
              <span>Attach support bundle</span>
            </label>
            <Button type="submit" variant="primary" disabled={submitting || !message.trim()}>
              Submit feedback
            </Button>
          </form>
        </>
      )}
    </div>
  );
}
