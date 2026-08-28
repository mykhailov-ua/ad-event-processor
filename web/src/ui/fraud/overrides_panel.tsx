import type { FormEvent } from 'react';
import * as auth from '../../helpers/auth.js';
import { canApplyFraudOverride } from '../../helpers/fraud_api.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import { FraudSubNav } from './fraud_sub_nav.js';
import styles from './fraud_shared.module.css';

export type OverridesPanelProps = {
  customerId: string;
  campaignId: string;
  ip: string;
  ipHash: string;
  error: unknown;
  formBusy: boolean;
  onCustomerApply: (customerId: string) => void;
  onCampaignIdChange: (value: string) => void;
  onIpChange: (value: string) => void;
  onIpHashChange: (value: string) => void;
  onSubmit: () => void;
};

export function OverridesPanel({
  customerId,
  campaignId,
  ip,
  ipHash,
  error,
  formBusy,
  onCustomerApply,
  onCampaignIdChange,
  onIpChange,
  onIpHashChange,
  onSubmit,
}: OverridesPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = canApplyFraudOverride(permissions);

  return (
    <div className={styles.root} data-testid="fraud-overrides-page">
      <PageChrome title="Fraud overrides" />
      <FraudSubNav customerId={customerId} />
      <p className={styles.intro}>
        Mark a false-positive scoring outcome. ML enforcement adds the IP to the fraud blacklist;
        it does not flip silent_reject_enabled on the campaign.
      </p>
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />
      {error ? <ErrorBlock error={error} fallbackTitle="Override request failed" /> : null}
      {canWrite ? (
        <form
          className={styles.formStack}
          onSubmit={(event: FormEvent) => {
            event.preventDefault();
            onSubmit();
          }}
        >
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Campaign ID (optional)</span>
            <input
              className={styles.textInput}
              value={campaignId}
              onChange={(event) => onCampaignIdChange(event.target.value)}
              placeholder="UUID"
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>IP address</span>
            <input
              className={styles.textInput}
              value={ip}
              onChange={(event) => onIpChange(event.target.value)}
              placeholder="1.2.3.4"
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>IP hash (32 hex, alternative)</span>
            <input
              className={styles.textInput}
              value={ipHash}
              onChange={(event) => onIpHashChange(event.target.value)}
              placeholder="abcdef0123456789abcdef0123456789"
            />
          </label>
          <p className={styles.helpText}>
            Provide either a plain IP or a 32-character ip_hash. At least one identifier is required.
          </p>
          <div className={styles.actions}>
            <Button
              type="submit"
              size="sm"
              variant="danger"
              disabled={formBusy || !customerId || (!ip.trim() && !ipHash.trim())}
            >
              Mark false positive
            </Button>
          </div>
        </form>
      ) : (
        <p className={styles.helpText}>
          Overrides require audit:write, campaigns:write, or shards:write.
        </p>
      )}
    </div>
  );
}
