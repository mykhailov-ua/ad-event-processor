import { toast } from 'sonner';

export function runCampaignListBulkAction(
  bulkBusy: boolean,
  allowed: boolean,
  hint: string,
  action?: () => void
) {
  if (bulkBusy) {
    toast.message('Bulk action in progress');
    return;
  }
  if (!allowed) {
    toast.message(hint);
    return;
  }
  action?.();
}
