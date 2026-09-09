import { AdminErrorDetails } from '@/shell/admin_error_details';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { formatAdminErrorDetails, userErrorMessage } from '@/lib/admin_error';
import { isValidationError, validationErrorMessage } from '@/lib/admin_validation_error';
import { cn } from '@/lib/utils';

type ErrorBlockProps = {
  title?: string;
  message?: string;
  error?: unknown;
  componentStack?: string;
};

export function ErrorBlock({
  title = 'Error',
  message,
  error,
  componentStack,
}: ErrorBlockProps) {
  const resolvedMessage =
    message ??
    (error != null && isValidationError(error)
      ? validationErrorMessage(error)
      : userErrorMessage(error, 'Request failed.'));
  const details =
    error != null || componentStack ? formatAdminErrorDetails(error, componentStack) : '';

  return (
    <div className={cn(uiSurfaces.messageError)} role="alert">
      <p className="font-semibold">{title}</p>
      <p>{resolvedMessage}</p>
      <AdminErrorDetails details={details} />
    </div>
  );
}
