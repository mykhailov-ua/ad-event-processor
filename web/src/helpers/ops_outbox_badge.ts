import { to } from '../lib/to.js';
import { api } from './api_client.js';
import { connectOpsLiveFeed } from './ops_live_feed.js';

export type OpsOutboxBadgeHandle = {
  destroy: () => void;
};

/**
 * Keep the Operations nav badge in sync via SSE with 30s poll fallback.
 */
export function startOpsOutboxBadge(onPending: (pending: number) => void): OpsOutboxBadgeHandle {
  let destroyed = false;

  /**
   * Pull outbox pending count from the ops summary endpoint.
   */
  async function pullSummary(): Promise<void> {
    const [res] = await to(api('/api/v1/ops/dashboard/summary'));
    if (destroyed || !res?.data) return;
    const data = res.data as { outbox_pending?: number };
    onPending(Number(data.outbox_pending) || 0);
  }

  const feed = connectOpsLiveFeed({
    pollMs: 30_000,
    onTick: (payload) => {
      if (payload.summary) {
        onPending(Number(payload.summary.outbox_pending) || 0);
      }
    },
    onPoll: () => { pullSummary(); },
  });

  pullSummary();

  return {
    destroy() {
      destroyed = true;
      feed.destroy();
    },
  };
}
