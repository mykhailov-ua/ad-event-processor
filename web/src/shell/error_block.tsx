import { AdminErrorDetails } from '@/shell/admin_error_details';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatAdminErrorDetails, userErrorMessage } from '@/lib/admin_error';

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
  const resolvedMessage = message ?? userErrorMessage(error, 'Request failed.');
  const details =
    error != null || componentStack
      ? formatAdminErrorDetails(error, componentStack)
      : '';

  return (
    <Card className="border-destructive/50 bg-destructive/5">
      <CardHeader className="pb-2">
        <CardTitle className="text-base text-destructive">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{resolvedMessage}</p>
        <AdminErrorDetails details={details} />
      </CardContent>
    </Card>
  );
}
