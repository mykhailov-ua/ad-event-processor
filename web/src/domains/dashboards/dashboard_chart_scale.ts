const MICRO_PER_USD = 1_000_000;
const NICE_STEP_MULTIPLIERS = [1, 2, 5, 10] as const;
const VOLUME_TICK_TARGET = 9;
const MONEY_TICK_TARGET = 9;
const EMPTY_VOLUME_STEP = 10;
const EMPTY_VOLUME_TOP = 60;
const EMPTY_MONEY_STEP_USD = 200;
const EMPTY_MONEY_TOP_USD = 1_000;

export type ChartAxisScale = {
  domain: [number, number];
  ticks: number[];
};

export function microToUsd(micro: number): number {
  return micro / MICRO_PER_USD;
}

export function usdToMicro(usd: number): number {
  return Math.round(usd * MICRO_PER_USD);
}

function buildTicksFromZero(step: number, top: number): number[] {
  const ticks: number[] = [];
  for (let value = 0; value <= top + 1e-9; value += step) {
    ticks.push(value);
  }
  return ticks;
}

function buildTicksBetween(step: number, floor: number, ceiling: number): number[] {
  const ticks: number[] = [];
  for (let value = floor; value <= ceiling + 1e-9; value += step) {
    ticks.push(value);
  }
  return ticks;
}

export function pickKeitaroStep(span: number, targetTicks: number): number {
  if (!Number.isFinite(span) || span <= 0) {
    return EMPTY_VOLUME_STEP;
  }
  const rough = span / targetTicks;
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  for (const multiplier of NICE_STEP_MULTIPLIERS) {
    const step = multiplier * magnitude;
    if (step * targetTicks >= span) {
      return step;
    }
  }
  return 10 * magnitude;
}

export function buildVolumeAxisScale(maxValue: number): ChartAxisScale {
  if (!Number.isFinite(maxValue) || maxValue <= 0) {
    return {
      domain: [0, EMPTY_VOLUME_TOP],
      ticks: buildTicksFromZero(EMPTY_VOLUME_STEP, EMPTY_VOLUME_TOP),
    };
  }
  const step = pickKeitaroStep(maxValue, VOLUME_TICK_TARGET);
  const top = Math.max(Math.ceil(maxValue / step) * step, step * 6);
  return {
    domain: [0, top],
    ticks: buildTicksFromZero(step, top),
  };
}

export function buildMoneyAxisScale(minMicro: number, maxMicro: number): ChartAxisScale {
  if (maxMicro <= 0 && minMicro >= 0) {
    return {
      domain: [0, usdToMicro(EMPTY_MONEY_TOP_USD)],
      ticks: buildTicksFromZero(usdToMicro(EMPTY_MONEY_STEP_USD), usdToMicro(EMPTY_MONEY_TOP_USD)),
    };
  }

  const minUsd = microToUsd(minMicro);
  const maxUsd = microToUsd(maxMicro);
  const spanUsd = maxUsd - minUsd;
  const stepUsd = pickKeitaroStep(spanUsd, MONEY_TICK_TARGET);

  let floorUsd: number;
  if (minUsd >= 0 && minUsd > stepUsd * 1.5) {
    floorUsd = Math.floor(minUsd / stepUsd) * stepUsd;
  } else if (minUsd < 0) {
    floorUsd = Math.floor(minUsd / stepUsd) * stepUsd;
  } else {
    floorUsd = 0;
  }

  let ceilingUsd = Math.ceil(maxUsd / stepUsd) * stepUsd;
  if (ceilingUsd <= floorUsd) {
    ceilingUsd = floorUsd + stepUsd;
  }

  const floorMicro = usdToMicro(floorUsd);
  const ceilingMicro = usdToMicro(ceilingUsd);
  const stepMicro = usdToMicro(stepUsd);

  return {
    domain: [floorMicro, ceilingMicro],
    ticks: buildTicksBetween(stepMicro, floorMicro, ceilingMicro),
  };
}

export function formatVolumeAxisTick(value: number): string {
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 }).format(value);
}

export function formatUsdAxisTick(micro: number): string {
  const usd = microToUsd(micro);
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 }).format(usd);
}

export function formatUsdTooltip(micro: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2,
  }).format(microToUsd(micro));
}

export function buildDateAxisTicks(labels: string[], targetLabels = 8): string[] {
  if (labels.length <= 1) {
    return labels;
  }
  const step = Math.max(1, Math.ceil(labels.length / targetLabels));
  const ticks: string[] = [];
  for (let index = 0; index < labels.length; index += step) {
    const label = labels[index]?.trim();
    if (label) {
      ticks.push(label);
    }
  }
  const lastLabel = labels.at(-1)?.trim();
  if (lastLabel && ticks.at(-1) !== lastLabel) {
    ticks.push(lastLabel);
  }
  return ticks;
}
