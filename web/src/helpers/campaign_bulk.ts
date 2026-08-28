import { to } from '../lib/to.js';
import { pauseCampaign, resumeCampaign } from './campaign_actions.js';
import { ConfirmCancelledError } from './confirm_ui.js';
import { parallelAll } from './request_multiplex.js';
import { pushToastMessage } from './toast_ui.js';

export async function runBulkCampaignAction(
  ids: string[],
  fn: (id: string) => Promise<void>
): Promise<Error | null> {
  const tasks = ids.map((id) => async () => {
    const [, err] = await to(fn(id));
    return err ?? null;
  });
  const results = await parallelAll(tasks, 3);
  for (let i = 0; i < results.length; i++) {
    const err = results[i];
    if (err instanceof ConfirmCancelledError) return err;
    if (err) {
      return err as Error;
    }
  }
  return null;
}

export function exportCampaignIds(ids: string[]): void {
  const text = ids.join('\n');
  if (ids.length === 0) {
    pushToastMessage({ title: 'Export', message: 'No campaigns selected' });
    return;
  }
  navigator.clipboard
    ?.writeText(text)
    .then(() => {
      pushToastMessage({ title: 'Exported', message: `${ids.length} campaign ID(s) copied` });
    })
    .catch(() => {
      pushToastMessage({ title: 'Export', message: text });
    });
}

export function bulkPauseCampaigns(ids: string[]): Promise<Error | null> {
  return runBulkCampaignAction(ids, pauseCampaign);
}

export function bulkResumeCampaigns(ids: string[]): Promise<Error | null> {
  return runBulkCampaignAction(ids, resumeCampaign);
}
