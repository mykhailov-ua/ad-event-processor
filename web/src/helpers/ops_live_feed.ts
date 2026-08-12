export type OpsFeedMode = 'stream' | 'poll';

export type OpsLiveFeedTick = {
  source: OpsFeedMode;
  summary?: Record<string, unknown>;
  generatedAt?: string;
};

export type OpsLiveFeedOpts = {
  onTick: (payload: OpsLiveFeedTick) => void;
  onPoll: () => void;
  onModeChange?: (mode: OpsFeedMode) => void;
  pollMs?: number;
};

export type OpsLiveFeedHandle = {
  destroy: () => void;
  mode: () => OpsFeedMode;
};

const STREAM_PATH = '/api/v1/ops/dashboard/stream';
const DEFAULT_POLL_MS = 30_000;

/**
 * Connect to the ops dashboard live feed (SSE) with parallel polling.
 */
export function connectOpsLiveFeed(opts: OpsLiveFeedOpts): OpsLiveFeedHandle {
  let destroyed = false;
  let mode: OpsFeedMode = 'poll';
  let es: EventSource | null = null;
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  const pollMs = opts.pollMs ?? DEFAULT_POLL_MS;

  /**
   * Switch feed mode and notify listeners.
   */
  function setMode(next: OpsFeedMode): void {
    if (mode === next) return;
    mode = next;
    opts.onModeChange?.(next);
  }

  /**
   * Start the poll fallback timer.
   */
  function startPoll(): void {
    if (pollTimer) return;
    pollTimer = setInterval(() => {
      if (!destroyed) opts.onPoll();
    }, pollMs);
  }

  /**
   * Open the SSE stream when EventSource is available.
   */
  function connectStream(): void {
    if (typeof EventSource === 'undefined') return;
    es = new EventSource(STREAM_PATH);
    es.addEventListener('dashboard', (event: MessageEvent<string>) => {
      if (destroyed) return;
      try {
        const payload = JSON.parse(event.data) as {
          data?: Record<string, unknown>;
          generated_at?: string;
        };
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
