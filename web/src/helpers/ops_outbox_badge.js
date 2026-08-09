import { to } from '../lib/to.js';
import { api } from './api_client.js';
import { connectOpsLiveFeed } from './ops_live_feed.js';

/**
 * Keep the Operations nav badge in sync via SSE with 30s poll fallback.
 *
 * @param {(pending: number) => void} onPending
 * @returns {{ destroy: () => void }}
 */
export function startOpsOutboxBadge(onPending) {
  let destroyed = false;

  async function pullSummary() {
    const [res] = await to(api('/api/v1/ops/dashboard/summary'));
    if (destroyed || !res?.data) return;
    onPending(Number(res.data.outbox_pending) || 0);
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
