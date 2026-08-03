/** @typedef {'stream' | 'poll'} OpsFeedMode */

const STREAM_PATH = '/api/v1/ops/dashboard/stream';
const DEFAULT_POLL_MS = 30_000;

/**
 * Connect to the ops dashboard live feed (SSE) with parallel polling.
 *
 * @param {{
 *   onTick: (payload: { source: OpsFeedMode, summary?: object, generatedAt?: string }) => void,
 *   onPoll: () => void,
 *   onModeChange?: (mode: OpsFeedMode) => void,
 *   pollMs?: number,
 * }} opts
 * @returns {{ destroy: () => void, mode: () => OpsFeedMode }}
 */
export function connectOpsLiveFeed(opts) {
  let destroyed = false;
  let mode = /** @type {OpsFeedMode} */ ('poll');
  /** @type {EventSource | null} */
  let es = null;
  /** @type {ReturnType<typeof setInterval> | null} */
  let pollTimer = null;
  const pollMs = opts.pollMs ?? DEFAULT_POLL_MS;

  /**
   * @param {OpsFeedMode} next
   */
  function setMode(next) {
    if (mode === next) return;
    mode = next;
    opts.onModeChange?.(next);
  }

  function startPoll() {
    if (pollTimer) return;
    pollTimer = setInterval(() => {
      if (!destroyed) opts.onPoll();
    }, pollMs);
  }

  function connectStream() {
    if (typeof EventSource === 'undefined') return;
    es = new EventSource(STREAM_PATH);
    es.addEventListener('dashboard', (event) => {
      if (destroyed) return;
      try {
        const payload = JSON.parse(event.data);
        setMode('stream');
        opts.onTick({
          source: 'stream',
          summary: payload.data,
          generatedAt: payload.generated_at,
        });
      } catch {
        setMode('poll');
      }
    });
    es.onerror = () => {
      if (destroyed) return;
      es?.close();
      es = null;
      setMode('poll');
    };
  }

  startPoll();
  connectStream();

  return {
    mode: () => mode,
    destroy() {
      destroyed = true;
      es?.close();
      es = null;
      if (pollTimer) clearInterval(pollTimer);
      pollTimer = null;
    },
  };
}
