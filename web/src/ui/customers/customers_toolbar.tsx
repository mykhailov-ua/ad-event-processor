import { useNavigate } from 'react-router-dom';
import { Button } from '../system/button.js';
import styles from './customers_directory.module.css';

export function CustomersToolbar() {
  const navigate = useNavigate();
  return (
    <div className={styles.toolbar}>
      <Button variant="secondary" size="sm" onClick={() => navigate('/billing')}>
        Open billing
      </Button>
    </div>
  );
}
