import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import styles from './audit_toolbar.module.css';

export type AuditToolbarProps = {
  exporting: boolean;
  onExport: () => void;
};

export function AuditToolbar({ exporting, onExport }: AuditToolbarProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  if (!can(permissions, 'audit:read')) {
    return null;
  }

  return (
    <div className={styles.root}>
      <Button variant="secondary" size="sm" disabled={exporting} onClick={onExport}>
        Export CSV
      </Button>
    </div>
  );
}
