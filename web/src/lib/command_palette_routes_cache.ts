import { listCommandPaletteRoutes, type CommandPaletteItem } from '@/api/command_palette_api';

let cachedRoutes: CommandPaletteItem[] | undefined;
let inflightRoutes: Promise<CommandPaletteItem[]> | undefined;

/**
 * Fetches command palette routes once per browser session; parallel mounts share inflight.
 */
export function fetchCommandPaletteRoutesCached(
  signal?: AbortSignal
): Promise<CommandPaletteItem[]> {
  if (cachedRoutes) {
    return Promise.resolve(cachedRoutes);
  }

  if (inflightRoutes) {
    return inflightRoutes;
  }

  inflightRoutes = listCommandPaletteRoutes(signal)
    .then((response) => {
      cachedRoutes = response.items ?? [];
      return cachedRoutes;
    })
    .finally(() => {
      inflightRoutes = undefined;
    });

  return inflightRoutes;
}
