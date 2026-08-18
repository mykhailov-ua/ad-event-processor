import * as storage from '../helpers/storage.js';
import { Button } from './button.js';

function listPendingIdempotencyScopes(): string[] {
  const out: string[] = [];
  try {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith('idem.pending.')) {
        out.push(key.slice('idem.pending.'.length));
      }
    }
  } catch {
    return [];
  }
  return out;
}

export function IdempotencyRecoveryBanner() {
  const pending = listPendingIdempotencyScopes();
  if (pending.length === 0) return null;

  return (
    <div className="alert-banner alert-banner--warning mb-4" role="status">
      <span className="alert-banner__text">
        {`Unfinished write operations (${pending.length}). Retry the action or clear pending keys.`}
      </span>
      <Button
        label="Clear pending"
        variant="secondary"
        size="sm"
        onClick={() => {
          storage.clearIdempotencyPendingAll();
          window.location.reload();
        }}
      />
    </div>
  );
}
