import { getMeta } from '@/api/platform_api';
import type { MetaResponse } from '@/api/types';

let cachedMeta: MetaResponse | undefined;
let inflightMeta: Promise<MetaResponse> | undefined;

/**
 * Fetches GET /meta once per browser session; MetaProvider, session bootstrap fallback,
 * and EulaGate share the same inflight promise.
 */
export function fetchMetaCached(signal?: AbortSignal): Promise<MetaResponse> {
  if (cachedMeta) {
    return Promise.resolve(cachedMeta);
  }

  if (inflightMeta) {
    return inflightMeta;
  }

  inflightMeta = getMeta(signal)
    .then((meta) => {
      cachedMeta = meta;
      return meta;
    })
    .finally(() => {
      inflightMeta = undefined;
    });

  return inflightMeta;
}

export function invalidateMetaCache(): void {
  cachedMeta = undefined;
  inflightMeta = undefined;
}
