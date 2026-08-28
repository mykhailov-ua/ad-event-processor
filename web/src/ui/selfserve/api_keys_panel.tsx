import { useState } from 'react';
import { createSelfServeApiKey } from '../../helpers/selfserve_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { StubBanner } from '../system/stub_banner.js';
import { ApiKeyOnceModal } from './api_key_once_modal.js';
import styles from './selfserve_shared.module.css';

export function ApiKeysPanel() {
  const [name, setName] = useState('');
  const [rawKey, setRawKey] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<unknown>(null);

  return (
    <div data-testid="selfserve-api-keys-panel">
      <PageChrome title="API keys" />
      <StubBanner
        title="List API not shipped"
        message="Only POST /api/v1/selfserve/api-keys exists. Existing keys cannot be listed from this page."
      />
      {error ? <ErrorBlock error={error} fallbackTitle="API key create failed" /> : null}
      <form
        className={styles.form}
        onSubmit={(e) => {
          e.preventDefault();
          setCreating(true);
          setError(null);
          void createSelfServeApiKey(name.trim())
            .then((res) => {
              if (res.raw_key) {
                setRawKey(res.raw_key);
              }
              pushToastMessage({
                title: 'API key created',
                message: 'Copy the key from the dialog. It is shown once.',
              });
              setName('');
            })
            .catch((err) => {
              if (err instanceof ConfirmCancelledError) return;
              setError(err);
            })
            .finally(() => setCreating(false));
        }}
      >
        <label className={styles.field}>
          <span className={styles.label}>Key name</span>
          <input
            className={styles.input}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        <Button type="submit" variant="primary" disabled={creating || !name.trim()}>
          {creating ? 'Creating...' : 'Create API key'}
        </Button>
      </form>
      {rawKey ? (
        <ApiKeyOnceModal
          rawKey={rawKey}
          onClose={() => {
            setRawKey(null);
          }}
        />
      ) : null}
    </div>
  );
}
