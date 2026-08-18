import { mapServiceError } from '../helpers/service_error.js';
import { Icon } from './icon.js';

export type ErrorBlockProps = {
  error: unknown;
  fallbackTitle?: string;
};

export function ErrorBlock({ error, fallbackTitle = 'Error' }: ErrorBlockProps) {
  if (!error) return null;
  const view = mapServiceError(error);
  return (
    <div className="error-page">
      <Icon name="alert-triangle" size={36} className="error-page__icon text-muted mb-3" />
      <div className="error-page__code">{String(view.status ?? '??')}</div>
      <div className="error-page__title">{view.title || fallbackTitle}</div>
      <div className="error-page__desc text-muted">{view.message}</div>
      {view.code && view.code !== view.message ? (
        <div className="text-muted text-xs mt-2">{view.code}</div>
      ) : null}
    </div>
  );
}
