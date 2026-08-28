export function parseIsoUnixSeconds(iso: string): number {
  if (!iso || iso.length < 19) return Number.NaN;
  const y = Number(iso.slice(0, 4));
  const mo = Number(iso.slice(5, 7));
  const d = Number(iso.slice(8, 10));
  const h = Number(iso.slice(11, 13));
  const mi = Number(iso.slice(14, 16));
  const s = Number(iso.slice(17, 19));
  return Date.UTC(y, mo - 1, d, h, mi, s) / 1000;
}
