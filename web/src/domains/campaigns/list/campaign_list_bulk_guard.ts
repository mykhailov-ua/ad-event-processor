import { actionGuardError, toastValidationError } from '@/lib/admin_validation_error';

export function runCampaignListBulkAction(
  bulkBusy: boolean,
  allowed: boolean,
  hint: string,
  action?: () => void
) {
  if (bulkBusy) {
    toastValidationError(actionGuardError('Bulk action in progress'));
    return;
  }
  if (!allowed) {
    toastValidationError(actionGuardError(hint));
    return;
  }
  action?.();
}
