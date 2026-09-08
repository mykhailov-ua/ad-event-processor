import { AdminErrorDetails } from '@/shell/admin_error_details';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { formatAdminErrorDetails, userErrorMessage } from '@/lib/admin_error';
import { cn } from '@/lib/utils';

type ErrorBlockProps = {
  title?: string;
  message?: string;
  error?: unknown;
  componentStack?: string;
  className?: string;
};

export function ErrorBlock({
  title = 'Error',
  message,
  error,
  componentStack,
  className,
}: ErrorBlockProps) {
  const resolvedMessage = message ?? userErrorMessage(error, 'Request failed.');
  const details =
    error != null || componentStack ? formatAdminErrorDetails(error, componentStack) : '';

  return (
    <div className={cn(uiSurfaces.messageError, className)} role="alert">
      <p className="m-0 text-base font-semibold">{title}</p>
      <p className="m-0 text-sm text-muted-foreground">{resolvedMessage}</p>
      <AdminErrorDetails details={details} />
    </div>
  );
}
