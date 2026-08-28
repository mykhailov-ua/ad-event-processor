import { useCallback, useEffect, useState } from 'react';
import { api } from '../helpers/api_client.js';
import { to } from '../lib/to.js';
import type { ResourceState } from '../lib/fetch_resource.js';

export function useResource<T>(
  url: string | null,
  options: { skip?: boolean } = {}
): ResourceState<T> & { reload: () => void } {
  const skip = options.skip ?? false;
  const [reloadToken, setReloadToken] = useState(0);
  const [state, setState] = useState<ResourceState<T>>({
    data: null,
    loading: !skip && Boolean(url),
    error: null,
  });

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    if (skip || !url) {
      setState({ data: null, loading: false, error: null });
      return undefined;
    }

    const ctrl = new AbortController();
    let cancelled = false;
    setState({ data: null, loading: true, error: null });

    void (async () => {
      const [result, err] = await to(api<T>(url, { signal: ctrl.signal }));
      if (cancelled) return;
      if (err) {
        if (err.name === 'AbortError') return;
        setState({ data: null, loading: false, error: err });
        return;
      }
      setState({ data: result?.data ?? null, loading: false, error: null });
    })();

    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [url, skip, reloadToken]);

  return { ...state, reload };
}
