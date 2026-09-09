import type { AdminValidationError } from '@/lib/admin_validation_error';
import { ErrorBlock } from '@/shell/error_block';

export type ValidationErrorBlockProps = {
  error: AdminValidationError | Error | undefined;
  title?: string;
};

export function ValidationErrorBlock({
  error,
  title = 'Check the form',
}: ValidationErrorBlockProps) {
  if (!error) {
    return null;
  }
  return <ErrorBlock title={title} error={error} />;
}
