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
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type AdminErrorPageProps = {
  kind: AdminErrorKind;
  error?: unknown;
  title?: string;
  message?: string;
  detail?: string;
  layout?: 'standalone' | 'embedded';
  componentStack?: string;
  onRetry?: () => void;
};

export function AdminErrorPage({
  kind,
  error,
  title,
  message,
  detail,
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
        'flex min-h-0 flex-1 items-center justify-center p-6',
        layout === 'standalone' && 'min-h-screen bg-background'
      )}
      role="alert"
    >
      <div
        className={cn(
          'w-full max-w-lg border border-border bg-card p-6 text-card-foreground',
          adminKit.panelRadius
        )}
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <p className="m-0 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {resolvedTitle}
            </p>
            <h1 className="m-0 text-xl font-semibold text-foreground">
              {kind === 'not-found' ? '404' : kind === 'forbidden' ? '403' : 'Error'}
            </h1>
            <p className="m-0 text-sm text-muted-foreground">{resolvedMessage}</p>
            {detail ? (
              <p className="m-0 text-sm text-muted-foreground">{detail}</p>
            ) : null}
            {devHint ? (
              <p className="m-0 font-mono text-xs text-muted-foreground">{devHint}</p>
            ) : null}
          </div>
          <div className="flex flex-wrap gap-2">
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
    <AdminErrorPage componentStack={componentStack} error={error} kind={kind} layout={layout} />
  );
}
