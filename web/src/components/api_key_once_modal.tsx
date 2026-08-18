import { pushToastMessage } from '../helpers/toast_ui.js';
import { Button } from './button.js';
import { Modal } from './modal.js';

export type ApiKeyOnceModalProps = {
  open: boolean;
  name: string;
  rawKey: string;
  expiresAt?: string;
  onClose: () => void;
};

export function ApiKeyOnceModal({ open, name, rawKey, expiresAt, onClose }: ApiKeyOnceModalProps) {
  const copyKey = () => {
    navigator.clipboard
      ?.writeText(rawKey)
      .then(() => {
        pushToastMessage({ title: 'Copied', message: 'API key copied to clipboard' });
      })
      .catch(() => {
        pushToastMessage({ title: 'Copy failed', message: 'Select the key and copy manually' });
      });
  };

  return (
    <Modal
      open={open}
      title="API key created"
      description={`Key "${name}" is shown once. Store it securely — it cannot be retrieved later.`}
      onClose={onClose}
      testId="api-key-once-modal"
      actions={
        <>
          <Button label="Copy to clipboard" variant="secondary" onClick={copyKey} />
          <Button label="I saved the key" variant="primary" onClick={onClose} />
        </>
      }
    >
      <label className="form-field" htmlFor="api-key-raw">
        Secret key
        <input
          id="api-key-raw"
          className="form-input font-mono"
          readOnly
          value={rawKey}
          onFocus={(e) => e.currentTarget.select()}
        />
      </label>
      {expiresAt ? <p className="text-muted text-sm">{`Expires: ${expiresAt}`}</p> : null}
    </Modal>
  );
}
