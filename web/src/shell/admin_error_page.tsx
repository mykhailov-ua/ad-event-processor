import { Link, useLocation } from 'react-router-dom';

import { AdminErrorDetails } from '@/shell/admin_error_details';
import { Button } from '@/components/ui/button';
import {
  adminErrorKindFromUnknown,
  adminErrorTitle,
  adminErrorUserMessage,
  formatAdminErrorDetails,
  shouldShowAdminErrorDetails,
  userErrorMessage,
  type AdminErrorKind,
} from '@/lib/admin_error';
import { cn } from '@/lib/utils';

export type AdminErrorPageProps = {
  kind: AdminErrorKind;
  error?: unknown;
  title?: string;
  message?: string;
  layout?: 'standalone' | 'embedded';
  componentStack?: string;
  onRetry?: () => void;
};

export function AdminErrorPage({
  kind,
  error,
  title,
  message,
  layout = 'embedded',
  componentStack,
  onRetry,
}: AdminErrorPageProps) {
  const location = useLocation();
  const resolvedTitle = title ?? adminErrorTitle(kind);
  const resolvedMessage =
    message ??
    (error != null
      ? userErrorMessage(error, adminErrorUserMessage(kind))
      : adminErrorUserMessage(kind));
  const details = formatAdminErrorDetails(error, componentStack);
  const devHint = shouldShowAdminErrorDetails()
    ? `Route: ${location.pathname}${location.search}`
    : undefined;

  function handleReload() {
    if (onRetry) {
      onRetry();
      return;
    }
    window.location.reload();
  }

  return (
    <div
      className={cn(
        'admin-error-page',
        layout === 'standalone' && 'admin-error-page--standalone',
      )}
      role="alert"
    >
      <div className="admin-error-page__card">
        <p className="admin-error-page__eyebrow">{resolvedTitle}</p>
        <h1 className="admin-error-page__title">
          {kind === 'not-found'
            ? '404'
            : kind === 'forbidden'
              ? '403'
              : 'Error'}
        </h1>
        <p className="admin-error-page__message">{resolvedMessage}</p>
        {devHint ? <p className="admin-error-page__route">{devHint}</p> : null}
        <div className="admin-error-page__actions">
          <Button type="button" variant="default" onClick={handleReload}>
            {onRetry ? 'Try again' : 'Reload page'}
          </Button>
          <Button asChild type="button" variant="outline">
            <Link to="/">Go home</Link>
          </Button>
        </div>
        <AdminErrorDetails details={details} />
      </div>
    </div>
  );
}

export function AdminErrorPageFromUnknown({
  error,
  layout = 'embedded',
  componentStack,
}: {
  error: unknown;
  layout?: 'standalone' | 'embedded';
  componentStack?: string;
}) {
  const kind = adminErrorKindFromUnknown(error);
  return (
    <AdminErrorPage
      componentStack={componentStack}
      error={error}
      kind={kind}
      layout={layout}
    />
  );
}
