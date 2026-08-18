import { isNetworkMacro } from '../models/traffic_source_templates.js';
import { defaultClickTemplate } from './tracking_link.js';

/**
 * Encode a query value unless it is a network macro token.
 */
export function encodeClickParamValue(value: string): string {
  if (!value) return '';
  return isNetworkMacro(value) ? value : encodeURIComponent(value);
}

/**
 * Resolve click path origin from a platform template or bare host.
 */
export function clickBaseURL(templateOrHost: string): string {
  const raw = String(templateOrHost || '').trim();
  if (!raw) {
    return defaultClickTemplate('').split('?')[0]!;
  }
  if (raw.includes('/click')) {
    try {
      const filled = raw.replaceAll('{campaign_id}', '00000000-0000-0000-0000-000000000000');
      const u = new URL(filled);
      return `${u.origin}${u.pathname}`;
    } catch {
      /* fall through */
    }
  }
  const host = raw
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '')
    .split('/')[0]!;
  return `https://${host}/click`;
}

export type ClickUrlOptions = {
  dmr?: boolean;
  utm?: Partial<
    Record<'utm_source' | 'utm_medium' | 'utm_campaign' | 'utm_term' | 'utm_content', string>
  >;
};

/**
 * Build a paste-ready click URL with campaign_id + template/manual params.
 * Network macros stay literal so buyers can paste into ad platforms.
 */
export function buildTemplatedClickURL(
  templateOrHost: string,
  campaignId: string,
  params: Record<string, string>,
  options?: ClickUrlOptions
): string {
  const base = clickBaseURL(templateOrHost);
  const parts: string[] = [`campaign_id=${encodeURIComponent(campaignId)}`];
  const keys = Object.keys(params);
  keys.sort((a, b) => a.localeCompare(b));
  for (const key of keys) {
    const val = params[key];
    if (val == null || val === '') continue;
    parts.push(`${encodeURIComponent(key)}=${encodeClickParamValue(val)}`);
  }
  if (options?.dmr) {
    parts.push('dmr=1');
  }
  if (options?.utm) {
    for (const [key, val] of Object.entries(options.utm)) {
      if (!val) continue;
      parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(val)}`);
    }
  }
  return `${base}?${parts.join('&')}`;
}
