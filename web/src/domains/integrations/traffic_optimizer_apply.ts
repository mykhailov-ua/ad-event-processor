import type { FlowPath } from '@/api/types';
import type { TrafficOptimizerDryRunResult, TrafficOptimizerRule } from '@/api/types';
import {
  flowPathsToVisualRows,
  normalizeEntityRefWeights,
  validateVisualPathWeights,
  visualRowsToFlowPaths,
} from '@/domains/creative/flow_path_model';

export type TrafficOptimizerApplyScope = 'lander' | 'offer';

export function ruleApplyScope(rule: TrafficOptimizerRule): TrafficOptimizerApplyScope | null {
  if (rule.scope === 'lander' || rule.scope === 'offer') {
    return rule.scope;
  }
  return null;
}

export function applyDryRunToFlowPaths(
  paths: FlowPath[],
  scope: TrafficOptimizerApplyScope,
  dryRun: TrafficOptimizerDryRunResult
): { ok: true; paths: FlowPath[] } | { ok: false; error: string } {
  const arms = dryRun.arms ?? [];
  if (arms.length === 0) {
    return { ok: false, error: 'Dry-run returned no weight suggestions.' };
  }
  const weightByEntity = new Map(
    arms.map((arm) => [arm.entity_id?.trim() ?? '', arm.proposed_weight ?? 0])
  );
  let matched = false;
  const rows = flowPathsToVisualRows(paths).map((row) => {
    if (scope === 'lander') {
      const landers = row.landers.map((ref) => {
        const entityId = ref.entity_id.trim();
        const proposed = weightByEntity.get(entityId);
        if (proposed == null || proposed <= 0) {
          return ref;
        }
        matched = true;
        return { ...ref, weight: proposed };
      });
      return { ...row, landers: normalizeEntityRefWeights(landers) };
    }
    const offers = row.offers.map((ref) => {
      const entityId = ref.entity_id.trim();
      const proposed = weightByEntity.get(entityId);
      if (proposed == null || proposed <= 0) {
        return ref;
      }
      matched = true;
      return { ...ref, weight: proposed };
    });
    return { ...row, offers: normalizeEntityRefWeights(offers) };
  });
  if (!matched) {
    return { ok: false, error: 'No matching lander or offer IDs in the target flow.' };
  }
  const validationError = validateVisualPathWeights(rows);
  if (validationError) {
    return { ok: false, error: validationError };
  }
  return { ok: true, paths: visualRowsToFlowPaths(rows) };
}
