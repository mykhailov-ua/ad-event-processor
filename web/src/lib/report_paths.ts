const REPORT_KEY_PATH_OVERRIDES: Record<string, string> = {
  'rtb-overview': '/api/v1/reports/rtb/overview',
  'rtb-no-bid-reasons': '/api/v1/reports/rtb/no-bid-reasons',
  'rtb-geo-device': '/api/v1/reports/rtb/geo-device',
};

export const EVIDENCE_PACK_REPORT_KEYS = new Set([
  'customer-fraud-evidence',
  'fraud-evidence-pack',
]);

export const EXPORT_ONLY_REPORT_KEYS = new Set(['fraud-evidence-pack-bulk']);

export function reportKeyToApiPath(key: string): string {
  const override = REPORT_KEY_PATH_OVERRIDES[key];
  if (override) {
    return override;
  }
  return `/api/v1/reports/${encodeURIComponent(key)}`;
}

export function defaultReportRange(defaultRange?: string): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to);
  const normalized = (defaultRange ?? '7d').trim().toLowerCase();
  const match = /^(\d+)([dh])$/.exec(normalized);

  if (match) {
    const amount = Number.parseInt(match[1], 10);
    if (match[2] === 'd') {
      from.setUTCDate(from.getUTCDate() - amount);
    } else {
      from.setUTCHours(from.getUTCHours() - amount);
    }
  } else {
    from.setUTCDate(from.getUTCDate() - 7);
  }

  return { from: from.toISOString(), to: to.toISOString() };
}
