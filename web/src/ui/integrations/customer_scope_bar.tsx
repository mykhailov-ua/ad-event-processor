import { useEffect, useState, type FormEvent } from 'react';
import { Button } from '../system/button.js';
import styles from './integrations_shared.module.css';

export type CustomerScopeBarProps = {
  customerId: string;
  onApply: (customerId: string) => void;
  label?: string;
};

export function CustomerScopeBar({
  customerId,
  onApply,
  label = 'Customer ID',
}: CustomerScopeBarProps) {
  const [draft, setDraft] = useState(customerId);

  useEffect(() => {
    setDraft(customerId);
  }, [customerId]);

  return (
    <form
      className={styles.scopeBar}
      onSubmit={(event: FormEvent) => {
        event.preventDefault();
        onApply(draft.trim());
      }}
    >
      <div className={styles.scopeRow}>
        <label className={styles.scopeField}>
          <span className={styles.scopeLabel}>{label}</span>
          <input
            className={styles.textInput}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="UUID"
            aria-label={label}
          />
        </label>
        <Button type="submit" variant="secondary">
          Apply
        </Button>
      </div>
    </form>
  );
}

export type CampaignScopeBarProps = {
  campaignId: string;
  onApply: (campaignId: string) => void;
};

export function CampaignScopeBar({ campaignId, onApply }: CampaignScopeBarProps) {
  const [draft, setDraft] = useState(campaignId);

  useEffect(() => {
    setDraft(campaignId);
  }, [campaignId]);

  return (
    <form
      className={styles.scopeBar}
      onSubmit={(event: FormEvent) => {
        event.preventDefault();
        onApply(draft.trim());
      }}
    >
      <div className={styles.scopeRow}>
        <label className={styles.scopeField}>
          <span className={styles.scopeLabel}>Campaign ID</span>
          <input
            className={styles.textInput}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="UUID"
            aria-label="Campaign ID"
          />
        </label>
        <Button type="submit" variant="secondary">
          Apply
        </Button>
      </div>
    </form>
  );
}
