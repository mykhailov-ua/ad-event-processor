/** Query param names accepted by tracker ingress cost parsing (`internal/domain/ingress_cost.go`). */
export type IngressCostParamName = 'cost' | 'cpc' | 'bid';

const INGRESS_COST_PARAMS: IngressCostParamName[] = ['cost', 'cpc', 'bid'];

/**
 * Returns true when `param` is a supported ingress cost query key.
 */
export function isIngressCostParam(param: string): param is IngressCostParamName {
  return (INGRESS_COST_PARAMS as string[]).includes(param);
}

/**
 * Default network macro placeholder for an ingress cost param (e.g. `{cost}`).
 */
export function ingressCostMacroPlaceholder(param: IngressCostParamName): string {
  return `{${param}}`;
}

/**
 * Resolves ingress cost param from campaign config or falls back to `cost`.
 */
export function resolveIngressCostParam(
  configured: string | undefined
): IngressCostParamName | null {
  const p = String(configured || '')
    .trim()
    .toLowerCase();
  if (!p) return null;
  return isIngressCostParam(p) ? p : null;
}
