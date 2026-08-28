import type { FraudPolicyPresetDTO } from '../../helpers/fraud_api.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { FraudSubNav } from './fraud_sub_nav.js';
import styles from './fraud_shared.module.css';

export type PresetsPanelProps = {
  presets: FraudPolicyPresetDTO[];
  loading: boolean;
  error: unknown;
};

export function PresetsPanel({ presets, loading, error }: PresetsPanelProps) {
  if (error && presets.length === 0 && !loading) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load fraud presets" />;
  }

  return (
    <div className={styles.root} data-testid="fraud-presets-page">
      <PageChrome title="Fraud policy presets" badge={loading ? null : <span>{presets.length} presets</span>} />
      <FraudSubNav />
      <p className={styles.intro}>
        Read-only tier thresholds (pass, suspect, IVT, block) applied by preset name. Updates are
        managed via the ops API.
      </p>
      <div className={styles.content}>
        {loading && presets.length === 0 ? (
          <PageSkeleton rows={3} columns={4} />
        ) : presets.length === 0 ? (
          <EmptyState message="No fraud policy presets returned." />
        ) : (
          <div className={styles.presetGrid} role="list">
            {presets.map((preset) => (
              <article key={preset.name} className={styles.presetCard} role="listitem">
                <h3 className={styles.presetName}>{preset.name}</h3>
                <div className={styles.presetRow}>
                  <span className={styles.presetLabel}>Pass</span>
                  <span className={styles.presetValue}>{preset.pass}</span>
                </div>
                <div className={styles.presetRow}>
                  <span className={styles.presetLabel}>Suspect</span>
                  <span className={styles.presetValue}>{preset.suspect}</span>
                </div>
                <div className={styles.presetRow}>
                  <span className={styles.presetLabel}>IVT</span>
                  <span className={styles.presetValue}>{preset.ivt}</span>
                </div>
                <div className={styles.presetRow}>
                  <span className={styles.presetLabel}>Block</span>
                  <span className={styles.presetValue}>{preset.block}</span>
                </div>
                {preset.updated_at ? (
                  <p className={styles.helpText}>Updated {preset.updated_at}</p>
                ) : null}
              </article>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
