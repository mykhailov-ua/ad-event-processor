import { Link } from 'react-router-dom';
import { Button } from '../ui/system/button.js';

export function NotFoundPage() {
  return (
    <div className="error-page">
      <div className="error-page__code">404</div>
      <div className="error-page__title">Page not found</div>
      <div className="error-page__desc text-muted">
        The requested route does not exist or is not implemented yet.
      </div>
      <Link to="/customers">
        <Button variant="secondary">
          Customers
        </Button>
      </Link>
    </div>
  );
}
