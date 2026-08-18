import { REPORT_DATE_PRESETS } from './date_presets.js';
import { boundCustomerId } from './buyer_session.js';
import type { TelegramReportQuery } from './tg_report_api.js';
import { tenantReportQueryString } from './tenant_url.js';

export type TelegramReportState = {
  customerInput: string;
  campaignInput: string;
  from: string;
  to: string;
  activePreset: string;
};

export type TelegramReportStateOpts = {
  sessionScoped?: boolean;
  user?: { customer_id?: string } | null;
};

export type TelegramCampaignOption = {
  id: string;
  name?: string;
};

export function createTelegramReportState(
  query: URLSearchParams,
  opts: TelegramReportStateOpts = {}
): TelegramReportState {
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];
  const sessionScoped = Boolean(opts.sessionScoped);
  const user = opts.user;
  return {
    customerInput: query.get('customer_id') || (sessionScoped && user ? boundCustomerId(user) : ''),
    campaignInput: query.get('campaign_id') || '',
    from: query.get('from') || preset.from(),
    to: query.get('to') || preset.to(),
    activePreset: query.get('preset') || preset.id,
  };
}

export function resolveTelegramCustomerId(
  state: TelegramReportState,
  sessionScoped: boolean,
  user: { customer_id?: string } | null | undefined
): string {
  return sessionScoped ? boundCustomerId(user) : state.customerInput.trim();
}

export function buildTelegramReportParams(
  state: TelegramReportState,
  sessionScoped: boolean,
  user: { customer_id?: string } | null | undefined
): TelegramReportQuery {
  const params: TelegramReportQuery = { from: state.from, to: state.to };
  const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
  if (customerId) params.customerId = customerId;
  const campaignId = state.campaignInput.trim();
  if (campaignId) params.campaignId = campaignId;
  return params;
}

export function applyTelegramPreset(
  state: TelegramReportState,
  preset: { id: string; from: () => string; to: () => string }
): TelegramReportState {
  return {
    ...state,
    activePreset: preset.id,
    from: preset.from(),
    to: preset.to(),
  };
}

export function syncTelegramReportUrl(path: string, state: TelegramReportState) {
  try {
    const qs = tenantReportQueryString({
      customer_id: state.customerInput.trim(),
      campaign_id: state.campaignInput.trim(),
      from: state.from,
      to: state.to,
      preset: state.activePreset,
    });
    window.history.replaceState(null, '', qs ? `${path}?${qs}` : path);
  } catch {}
}

export function telegramReportHref(path: string, state: TelegramReportState): string {
  const qs = tenantReportQueryString({
    customer_id: state.customerInput.trim(),
    campaign_id: state.campaignInput.trim(),
    from: state.from,
    to: state.to,
    preset: state.activePreset,
  });
  return qs ? `${path}?${qs}` : path;
}
