import { api } from '../helpers/api_client.js';
import { to } from './to.js';

export type JsonObject = Record<string, unknown>;

export type ResourceState<T = JsonObject> = {
  data: T | null;
  loading: boolean;
  error: unknown | null;
};

export type ResourceHandle = {
  reload: () => Promise<void>;
  destroy: () => void;
};

export function createResource<T = JsonObject>(
  getUrl: () => string | null | undefined,
  opts: {
    skip?: () => boolean;
    onUpdate: (state: ResourceState<T>) => void;
  }
): ResourceHandle {
  const skip = opts.skip ?? (() => false);
  let abort: AbortController | null = null;
  let gen = 0;
  let destroyed = false;

  async function load(): Promise<void> {
    if (destroyed) return;
    const url = getUrl();
    if (!url || skip()) {
      opts.onUpdate({ data: null, loading: false, error: null });
      return;
    }
    if (abort) abort.abort();
    const ctrl = new AbortController();
    abort = ctrl;
    const id = ++gen;
    opts.onUpdate({ data: null, loading: true, error: null });
    const [result, err] = await to(api<T>(url, { signal: ctrl.signal }));
    if (destroyed || id !== gen) return;
    if (err) {
      if (err.name === 'AbortError') return;
      opts.onUpdate({ data: null, loading: false, error: err });
      return;
    }
    opts.onUpdate({ data: result?.data ?? null, loading: false, error: null });
  }

  void load();

  return {
    reload: load,
    destroy() {
      destroyed = true;
      abort?.abort();
      gen += 1;
    },
  };
}
