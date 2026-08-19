import { isNetworkMacro } from '../models/traffic_source_templates.js';
import { defaultClickTemplate } from './tracking_link.js';


export function encodeClickParamValue(value: string): string {
  if (!value) return '';
  return isNetworkMacro(value) ? value : encodeURIComponent(value);
}


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
      
    }
  }
  const host = raw
    .replace(/^https?:\/\    .replace(/\/+$/, '')
    .split('/')[0]!;
  return `https://${host}/click`;
}

export type ClickUrlOptions = {
  dmr?: boolean;
  utm?: Partial<
    Record<'utm_source' | 'utm_medium' | 'utm_campaign' | 'utm_term' | 'utm_content', string>
  >;
};


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
