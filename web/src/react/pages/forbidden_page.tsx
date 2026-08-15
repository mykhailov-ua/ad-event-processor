import { useNavigate } from 'react-router-dom';
import { buyerEmptyCopy } from '../../models/empty_state.js';
import { Button } from '../components/button.js';

/**
 * 403 page for routes the user cannot access.
 */
export function ForbiddenPage() {
  const navigate = useNavigate();
  const copy = buyerEmptyCopy('forbidden');

  return (
    <main>
      <h1>{copy.title}</h1>
      <p>{copy.description}</p>
      <Button
        label={copy.actionLabel ?? 'Continue'}
        variant="secondary"
        onClick={() => navigate(copy.actionHref ?? '/campaigns')}
      />
    </main>
  );
}
