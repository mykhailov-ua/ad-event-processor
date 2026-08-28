import styles from './audit_directory.module.css';

export type AuditFilterProps = {
  redactPii: boolean;
  onChange: (redactPii: boolean) => void;
};

export function AuditFilter({ redactPii, onChange }: AuditFilterProps) {
  return (
    <div className={styles.filters}>
      <label className={styles.filterLabel}>
        <input
          type="checkbox"
          checked={redactPii}
          onChange={(event) => onChange(event.target.checked)}
        />
        <span>Redact PII in changes/metadata</span>
      </label>
    </div>
  );
}
