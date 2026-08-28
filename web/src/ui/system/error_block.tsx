import { mapServiceError } from '../../helpers/service_error.js';
import { Button } from './button.js';
import styles from './error_block.module.css';

export type ErrorBlockProps = {
  error: unknown;
  fallbackTitle?: string;
  onRetry?: () => void;
};

export function ErrorBlock({ error, fallbackTitle = 'Error', onRetry }: ErrorBlockProps) {
  if (!error) return null;
  const view = mapServiceError(error);
  return (
    <div className={styles.root} role="alert">
      <p className={styles.title}>{view.title || fallbackTitle}</p>
      <p className={styles.message}>{view.message}</p>
      {view.code && view.code !== view.message ? (
        <p className={styles.code}>{view.code}</p>
      ) : null}
      {onRetry ? (
        <div className={styles.actions}>
          <Button type="button" variant="secondary" onClick={onRetry}>
            Retry
          </Button>
        </div>
      ) : null}
    </div>
  );
}
