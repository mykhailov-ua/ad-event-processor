import { useState } from 'react';
import type { CampaignBulkAction } from '../../helpers/campaigns_api.js';
import { bulkMutateCampaigns } from '../../helpers/campaigns_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { Button } from '../system/button.js';
import styles from './campaigns_bulk_bar.module.css';

export type CampaignsBulkBarProps = {
  selectedIds: string[];
  onClear: () => void;
  onSuccess: () => void;
};

export function CampaignsBulkBar({ selectedIds, onClear, onSuccess }: CampaignsBulkBarProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (selectedIds.length === 0) {
    return null;
  }

  const runBulk = async (action: CampaignBulkAction) => {
    setBusy(true);
    setError(null);
    try {
      await bulkMutateCampaigns(action, selectedIds);
      pushToastMessage({
        title: action === 'pause' ? 'Paused' : 'Resumed',
        message: `${selectedIds.length} campaign(s) updated`,
      });
      onClear();
      onSuccess();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err instanceof Error ? err.message : 'Bulk action failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.root} role="region" aria-label="Bulk actions">
      <span className={styles.count}>{selectedIds.length} selected</span>
      <Button
        variant="secondary"
       
        disabled={busy}
        onClick={() => void runBulk('pause')}
      >
        Pause
      </Button>
      <Button
        variant="secondary"
       
        disabled={busy}
        onClick={() => void runBulk('resume')}
      >
        Resume
      </Button>
      <Button variant="secondary" disabled={busy} onClick={onClear}>
        Clear
      </Button>
      {error ? <span className={styles.count}>{error}</span> : null}
    </div>
  );
}
