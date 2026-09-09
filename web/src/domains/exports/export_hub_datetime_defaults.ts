import { formatDatetimeLocalValue } from '@/lib/datetime_range';

export function defaultExportHubDatetimeRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to.getTime() - 24 * 60 * 60 * 1000);
  return {
    from: formatDatetimeLocalValue(from),
    to: formatDatetimeLocalValue(to),
  };
}
