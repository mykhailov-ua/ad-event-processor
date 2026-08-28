import { useState } from 'react';
import { to } from '../lib/to.js';
import { createApiKey } from '../helpers/api_keys_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { ApiKeyOnceModal } from './api_key_once_modal.js';
import { Button } from './button.js';

export type CustomerApiKeysSectionProps = {
  canCreate: boolean;
};

export function CustomerApiKeysSection({ canCreate }: CustomerApiKeysSectionProps) {
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [keyModal, setKeyModal] = useState<{
    name: string;
    rawKey: string;
    expiresAt?: string;
  } | null>(null);

  async function onCreate() {
    if (!canCreate || busy) return;
    const trimmed = name.trim();
    if (!trimmed) {
      setError('Key name is required');
      return;
    }
    setBusy(true);
    setError(null);
    const [data, err] = await to(createApiKey(trimmed));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    setName('');
    if (data?.raw_key) {
      setKeyModal({
        name: data.name ?? trimmed,
        rawKey: data.raw_key,
        expiresAt: data.expires_at,
      });
    } else {
      pushToastMessage({ title: 'Key created', message: 'No raw key returned by API' });
    }
  }

  return (
    <>
      <section className="section-block section-card stack">
        <h2 className="subsection-title">API keys</h2>
        <p className="text-muted text-sm">
          Create integration keys for tracking and automation. The secret is shown once after
          creation.
        </p>
        {error ? <p className="text-danger text-sm">{error}</p> : null}
        {canCreate ? (
          <div className="flex items-end gap-2 flex-wrap">
            <label
              className="form-field flex-1"
              htmlFor="api-key-name"
              style={{ minWidth: '12rem' }}
            >
              Key name
              <input
                id="api-key-name"
                className="form-input"
                placeholder="e.g. Keitaro postback"
                value={name}
                disabled={busy}
                onInput={(e) => setName(e.currentTarget.value)}
              />
            </label>
            <Button
              label={busy ? 'Creating...' : 'Create API key'}
              variant="primary"
              size="sm"
              loading={busy}
              disabled={busy || !name.trim()}
              onClick={() => void onCreate()}
            />
          </div>
        ) : (
          <p className="text-muted text-sm">You need campaigns:write to create API keys.</p>
        )}
      </section>

      <ApiKeyOnceModal
        open={keyModal !== null}
        name={keyModal?.name ?? ''}
        rawKey={keyModal?.rawKey ?? ''}
        expiresAt={keyModal?.expiresAt}
        onClose={() => setKeyModal(null)}
      />
    </>
  );
}
