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
        'flex min-h-0 flex-1 items-center justify-center p-6',
        layout === 'standalone' && 'min-h-screen bg-zinc-50 dark:bg-zinc-950',
      )}
      role="alert"
    >
      <div className="w-full max-w-lg rounded-lg border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
        <p className="text-xs font-semibold uppercase tracking-wide text-zinc-500">{resolvedTitle}</p>
        <h1 className="mt-2 text-xl font-semibold">
          {kind === 'not-found'
            ? '404'
            : kind === 'forbidden'
              ? '403'
              : 'Error'}
        </h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">{resolvedMessage}</p>
        {devHint ? <p className="mt-2 font-mono text-xs text-zinc-500">{devHint}</p> : null}
        <div className="mt-6 flex flex-wrap gap-2">
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
