import { listCommandPaletteRecents, type CommandPaletteItem } from '@/api/command_palette_api';

const cachedByCustomerId = new Map<string, CommandPaletteItem[]>();
const inflightByCustomerId = new Map<string, Promise<CommandPaletteItem[]>>();

/**
 * Fetches command palette recents once per browser session per customer; parallel opens share inflight.
 */
export function fetchCommandPaletteRecentsCached(
  customerId: string,
  signal?: AbortSignal
): Promise<CommandPaletteItem[]> {
  const cached = cachedByCustomerId.get(customerId);
  if (cached) {
    return Promise.resolve(cached);
  }

  const inflight = inflightByCustomerId.get(customerId);
  if (inflight) {
    return inflight;
  }

  const request = listCommandPaletteRecents(customerId, signal)
    .then((response) => {
      const items = response.items ?? [];
      cachedByCustomerId.set(customerId, items);
      return items;
    })
    .finally(() => {
      inflightByCustomerId.delete(customerId);
    });

  inflightByCustomerId.set(customerId, request);
  return request;
}

export function invalidateCommandPaletteRecentsCache(customerId?: string): void {
  if (customerId) {
    cachedByCustomerId.delete(customerId);
    inflightByCustomerId.delete(customerId);
  } else {
    cachedByCustomerId.clear();
    inflightByCustomerId.clear();
  }
}
