import { useNavigate } from 'react-router-dom';
import { Button } from '../ui/system/button.js';

export function ForbiddenPage() {
  const navigate = useNavigate();

  return (
    <main className="error-page">
      <h1>Access denied</h1>
      <p className="text-muted">You do not have permission for this page.</p>
      <Button variant="secondary" onClick={() => navigate('/customers')}>
        Back to customers
      </Button>
    </main>
  );
}
