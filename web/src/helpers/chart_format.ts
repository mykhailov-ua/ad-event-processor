export function formatChartTick(n: number): string {
  const abs = Math.abs(n);
  if (abs >= 1_000_000) return `${(n / 1_000_000).toFixed(abs >= 10_000_000 ? 0 : 1)}M`;
  if (abs >= 10_000) return `${Math.round(n / 1000)}k`;
  if (abs >= 1000) return `${(n / 1000).toFixed(1)}k`;
  if (Number.isInteger(n)) return String(n);
  return n.toFixed(abs < 10 ? 1 : 0);
}

const axisDate = new Date(0);

export function formatChartAxisTime(ts: number, rangeMs = 24 * 60 * 60 * 1000): string {
  axisDate.setTime(ts);
  const hh = String(axisDate.getHours()).padStart(2, '0');
  const mm = String(axisDate.getMinutes()).padStart(2, '0');
  if (rangeMs <= 60 * 60 * 1000) {
    const ss = String(axisDate.getSeconds()).padStart(2, '0');
    return `${hh}:${mm}:${ss}`;
  }
  return `${hh}:${mm}`;
}

export function formatChartTime(ts: number): string {
  const d = new Date(ts);
  const y = d.getFullYear();
  const mo = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  const ss = String(d.getSeconds()).padStart(2, '0');
  return `${y}-${mo}-${day} ${hh}:${mm}:${ss}`;
}

export function formatClockTime(ts: number): string {
  if (!Number.isFinite(ts) || ts <= 0) return '-';
  return formatChartTime(ts);
}

export function formatRefreshCountdown(msRemaining: number): string {
  if (msRemaining <= 0) return '0:00';
  const sec = Math.ceil(msRemaining / 1000);
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
}
