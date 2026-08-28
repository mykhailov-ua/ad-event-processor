import { useNavigate } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import styles from './campaigns_directory.module.css';

export function CampaignsToolbar() {
  const navigate = useNavigate();
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');

  if (!canWrite) {
    return null;
  }

  return (
    <div className={styles.toolbar}>
      <Button variant="primary" onClick={() => navigate('/campaigns/wizard')}>
        Create campaign
      </Button>
      <Button variant="secondary" onClick={() => navigate('/campaigns/migrate')}>
        Migrate
      </Button>
    </div>
  );
}
