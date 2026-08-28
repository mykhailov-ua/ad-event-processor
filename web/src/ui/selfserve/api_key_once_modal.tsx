import { Button } from '../system/button.js';
import styles from './api_key_once_modal.module.css';

export type ApiKeyOnceModalProps = {
  rawKey: string;
  onClose: () => void;
};

export function ApiKeyOnceModal({ rawKey, onClose }: ApiKeyOnceModalProps) {
  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(rawKey);
    } catch {
      /* clipboard may be unavailable */
    }
  };

  return (
    <div className={styles.overlay} role="dialog" aria-modal="true" aria-labelledby="api-key-once-title">
      <div className={styles.dialog}>
        <h2 id="api-key-once-title" className={styles.title}>
          API key created
        </h2>
        <p className={styles.message}>
          Copy this key now. It is not stored in the UI and cannot be listed later.
        </p>
        <div className={styles.keyBox}>{rawKey}</div>
        <div className={styles.actions}>
          <Button type="button" variant="secondary" onClick={() => void onCopy()}>
            Copy
          </Button>
          <Button type="button" variant="primary" onClick={onClose}>
            Done
          </Button>
        </div>
      </div>
    </div>
  );
}
