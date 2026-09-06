import { PERF_TOLERANCE_RATIO, type PerfBudget } from '@/lib/perf/budgets';

export type BenchResult = {
  name: string;
  medianMs: number;
  budgetMs: number;
  limitMs: number;
  iterations: number;
  batchSize: number;
};

function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 0) {
    return (sorted[mid - 1] + sorted[mid]) / 2;
  }
  return sorted[mid];
}

export function measureMedianMs(fn: () => void, iterations: number): number {
  const samples: number[] = [];
  for (let i = 0; i < iterations; i += 1) {
    const t0 = performance.now();
    fn();
    samples.push(performance.now() - t0);
  }
  return median(samples);
}

export function runPerfBudget(budget: PerfBudget, fn: () => void): BenchResult {
  const batchSize = budget.batchSize ?? 1;
  for (let i = 0; i < budget.warmupIterations; i += 1) {
    fn();
  }

  const medianMs = measureMedianMs(fn, budget.measureIterations);
  const perUnitMs = batchSize > 1 ? medianMs / batchSize : medianMs;
  const limitMs = budget.medianMs * PERF_TOLERANCE_RATIO;

  return {
    name: budget.name,
    medianMs: perUnitMs,
    budgetMs: budget.medianMs,
    limitMs,
    iterations: budget.measureIterations,
    batchSize,
  };
}

export function assertPerfBudget(result: BenchResult): void {
  if (result.medianMs > result.limitMs) {
    throw new Error(
      `${result.name}: median ${result.medianMs.toFixed(3)} ms exceeds budget ${result.budgetMs} ms (+${Math.round((PERF_TOLERANCE_RATIO - 1) * 100)}% => ${result.limitMs.toFixed(3)} ms)`
    );
  }
}

export function formatBenchResult(result: BenchResult): string {
  return `${result.name}: ${result.medianMs.toFixed(3)} ms (budget ${result.budgetMs} ms, limit ${result.limitMs.toFixed(3)} ms)`;
}
