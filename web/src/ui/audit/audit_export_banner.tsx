import { Button } from '../system/button.js';
import styles from './audit_export_banner.module.css';

export type AuditExportBannerProps = {
  truncated: boolean;
  nextCursor: string | null;
  onContinue?: () => void;
  onDismiss: () => void;
};

export function AuditExportBanner({
  truncated,
  nextCursor,
  onContinue,
  onDismiss,
}: AuditExportBannerProps) {
  if (!truncated) return null;

  return (
    <div className={styles.root} role="status">
      <span>Export was truncated. More rows are available.</span>
      <div className={styles.actions}>
        {nextCursor && onContinue ? (
          <Button variant="secondary" onClick={onContinue}>
            Continue export
          </Button>
        ) : null}
        <Button variant="secondary" onClick={onDismiss}>
          Dismiss
        </Button>
      </div>
    </div>
  );
}
