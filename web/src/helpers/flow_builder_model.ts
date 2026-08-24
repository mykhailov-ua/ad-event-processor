import type { FlowPathDTO, FlowPathLanderRef, FlowPathOfferRef } from './flows_api.js';

/** Default weight for a new flow path row. */
export const DEFAULT_FLOW_ROW_WEIGHT = 100;

/**
 * Build an empty flow path with optional default lander and offer selections.
 * @param landerId - Optional lander UUID for the first row.
 * @param offerId - Optional offer UUID for the first row.
 */
export function emptyFlowPath(landerId = '', offerId = ''): FlowPathDTO {
  return {
    weight: DEFAULT_FLOW_ROW_WEIGHT,
    landers: landerId ? [{ lander_id: landerId, weight: DEFAULT_FLOW_ROW_WEIGHT }] : [],
    offers: offerId ? [{ offer_id: offerId, weight: DEFAULT_FLOW_ROW_WEIGHT }] : [],
  };
}

/**
 * Move an array item up or down by one index.
 * @param items - Source list (not mutated).
 * @param index - Item index to move.
 * @param delta - -1 for up, +1 for down.
 */
export function moveFlowBuilderItem<T>(items: T[], index: number, delta: number): T[] {
  const target = index + delta;
  if (target < 0 || target >= items.length) {
    return items;
  }
  const next = items.slice();
  const [row] = next.splice(index, 1);
  next.splice(target, 0, row);
  return next;
}

/**
 * Append a lander row to a path when the lander id is set.
 * @param path - Flow path to extend.
 * @param landerId - Lander UUID.
 */
export function appendLanderRow(path: FlowPathDTO, landerId: string): FlowPathDTO {
  if (!landerId) return path;
  const row: FlowPathLanderRef = { lander_id: landerId, weight: DEFAULT_FLOW_ROW_WEIGHT };
  return { ...path, landers: [...path.landers, row] };
}

/**
 * Append an offer row to a path when the offer id is set.
 * @param path - Flow path to extend.
 * @param offerId - Offer UUID.
 */
export function appendOfferRow(path: FlowPathDTO, offerId: string): FlowPathDTO {
  if (!offerId) return path;
  const row: FlowPathOfferRef = { offer_id: offerId, weight: DEFAULT_FLOW_ROW_WEIGHT };
  return { ...path, offers: [...path.offers, row] };
}

/**
 * Sum path weights for display in the builder header.
 * @param paths - Flow paths from the editor state.
 */
export function totalFlowPathWeight(paths: FlowPathDTO[]): number {
  return paths.reduce((sum, path) => sum + (Number.isFinite(path.weight) ? path.weight : 0), 0);
}
