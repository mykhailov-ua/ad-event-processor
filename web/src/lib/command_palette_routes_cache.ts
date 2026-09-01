import { listCommandPaletteRoutes, type CommandPaletteItem } from '@/api/command_palette_api';

let cachedRoutes: CommandPaletteItem[] | undefined;
let inflightRoutes: Promise<CommandPaletteItem[]> | undefined;

/**
 * Fetches command palette routes once per browser session; recents stay fresh on each open.
 */
export function fetchCommandPaletteRoutesCached(
  signal?: AbortSignal,
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
