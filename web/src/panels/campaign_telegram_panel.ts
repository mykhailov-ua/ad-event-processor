import type { ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderFormField } from '../ui/form_field.js';
import { renderSectionCard } from '../ui/section_card.js';
import { renderButton } from '../ui/button.js';
import { renderEmptyState } from '../ui/data_table.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import {
  createTelegramDeeplink,
  createTelegramPostback,
  deleteTelegramPostback,
  fetchTelegramBot,
  fetchTelegramPostbacks,
  saveTelegramBot,
  testTelegramPostback,
  type TelegramDeeplinkDTO,
  type TelegramPostbackDTO,
} from '../helpers/tg_admin_api.js';

export type CampaignTelegramPanelOpts = {
  campaignId: string;
  canWrite: boolean;
};

type TelegramBotForm = {
  bot_id: string;
  bot_token: string;
  webhook_url: string;
  mini_app_url: string;
  secret_token: string;
  auth_date_ttl: string;
};

type TelegramPanelState = {
  loading: boolean;
  error: unknown | null;
  configured: boolean;
  bot: TelegramBotForm;
  tokenInput: string;
  secretInput: string;
  postbacks: TelegramPostbackDTO[];
  newPostbackUrl: string;
  deeplink: TelegramDeeplinkDTO | null;
  utmSource: string;
  utmCampaign: string;
  saving: boolean;
  postbackBusy: boolean;
};

/**
 * Mask a secret for display (last 4 chars visible).
 *
 * @param {string} value
 * @returns {string}
 */
function maskSecret(value: any) {
  if (!value) return '—';
  if (value.length <= 4) return '••••';
  return `••••${value.slice(-4)}`;
}

/**
 * Copy text to clipboard with toast feedback.
 *
 * @param {string} text
 */
async function copyText(text: any) {
  try {
    await navigator.clipboard.writeText(text);
    pushToastMessage({ title: 'Copied', message: 'Copied to clipboard' });
  } catch {
    pushToastMessage({ title: 'Copy failed', message: 'Could not access clipboard' });
  }
}

/**
 * Mount Telegram Mini App configuration for a campaign.
 *
 * @param {HTMLElement} container
 * @param {CampaignTelegramPanelOpts} opts
 * @returns {{ destroy: () => void }}
 */
export function mountCampaignTelegramPanel(container: HTMLElement, opts: CampaignTelegramPanelOpts): ViewHandle {
  let destroyed = false;
  const { campaignId, canWrite } = opts;
  const gate = createInFlightGuard();

  const state: TelegramPanelState = {
    loading: true,
    error: null,
    configured: false,
    bot: {
      bot_id: '',
      bot_token: '',
      webhook_url: '',
      mini_app_url: '',
      secret_token: '',
      auth_date_ttl: '300',
    },
    tokenInput: '',
    secretInput: '',
    postbacks: [],
    newPostbackUrl: '',
    deeplink: null,
    utmSource: '',
    utmCampaign: '',
    saving: false,
    postbackBusy: false,
  };

  /** @type {string} */
  let storedToken = '';
  /** @type {string} */
  let storedSecret = '';

  async function load() {
    state.loading = true;
    state.error = null;
    render();
    const [bot, postbacks] = await Promise.all([
      to(fetchTelegramBot(campaignId)),
      to(fetchTelegramPostbacks(campaignId)),
    ]);
    if (destroyed) return;
    state.loading = false;
    if (bot[1]) {
      state.error = bot[1];
      render();
      return;
    }
    if (postbacks[1]) {
      state.error = postbacks[1];
      render();
      return;
    }
    const botData = bot[0];
    if (botData) {
      state.configured = true;
      storedToken = botData.bot_token ?? '';
      storedSecret = botData.secret_token ?? '';
      state.bot = {
        bot_id: String(botData.bot_id ?? ''),
        bot_token: '',
        webhook_url: botData.webhook_url ?? '',
        mini_app_url: botData.mini_app_url ?? '',
        secret_token: '',
        auth_date_ttl: String(botData.auth_date_ttl ?? 300),
      };
    } else {
      state.configured = false;
      storedToken = '';
      storedSecret = '';
    }
    state.postbacks = postbacks[0] ?? [];
    render();
  }

  async function handleSaveBot(e: Event) {
    e.preventDefault();
    if (!canWrite || !gate.tryAcquire()) return;
    state.saving = true;
    render();
    const token = state.tokenInput.trim() || storedToken;
    const secret = state.secretInput.trim() || storedSecret;
    const [, err] = await to(saveTelegramBot(campaignId, {
      bot_id: Number(state.bot.bot_id) || 0,
      bot_token: token,
      webhook_url: state.bot.webhook_url,
      mini_app_url: state.bot.mini_app_url,
      secret_token: secret,
      auth_date_ttl: Number(state.bot.auth_date_ttl) || 300,
    }));
    if (destroyed) {
      gate.release();
      return;
    }
    state.saving = false;
    gate.release();
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      render();
      return;
    }
    state.tokenInput = '';
    state.secretInput = '';
    pushToastMessage({ title: 'Saved', message: 'Telegram bot configuration updated' });
    load();
  }

  async function handleCreateDeeplink() {
    if (!canWrite || !gate.tryAcquire()) return;
    const [dl, err] = await to(createTelegramDeeplink(campaignId, {
      utm_source: state.utmSource,
      utm_campaign: state.utmCampaign,
    }));
    gate.release();
    if (destroyed) return;
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    state.deeplink = dl;
    pushToastMessage({ title: 'Deeplink created', message: 'Bridge token generated' });
    render();
  }

  async function handleAddPostback(e: Event) {
    e.preventDefault();
    const url = state.newPostbackUrl.trim();
    if (!url || !canWrite || !gate.tryAcquire()) return;
    state.postbackBusy = true;
    render();
    const [, err] = await to(createTelegramPostback(campaignId, url));
    if (destroyed) {
      gate.release();
      return;
    }
    state.postbackBusy = false;
    gate.release();
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      render();
      return;
    }
    state.newPostbackUrl = '';
    pushToastMessage({ title: 'Postback added', message: 'S2S postback URL saved' });
    load();
  }

  async function handleTestPostback(id: any) {
    if (!canWrite || !gate.tryAcquire()) return;
    const [, err] = await to(testTelegramPostback(id));
    gate.release();
    if (destroyed) return;
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    pushToastMessage({ title: 'Test sent', message: 'Postback URL responded OK' });
  }

  async function handleDeletePostback(id: any) {
    if (!canWrite || !gate.tryAcquire()) return;
    const [, err] = await to(deleteTelegramPostback(id));
    gate.release();
    if (destroyed) return;
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    pushToastMessage({ title: 'Deleted', message: 'Postback removed' });
    load();
  }

  function webhookIngestUrl() {
    const botId = state.bot.bot_id.trim();
    if (!botId) return '';
    return `${window.location.origin}/api/v1/telegram/webhook/${botId}`;
  }

  function clickBridgeUrl(token: any) {
    const q = new URLSearchParams({
      campaign_id: campaignId,
      bridge_token: token,
    });
    return `${window.location.origin}/tg/click?${q.toString()}`;
  }

  function renderBotSection() {
    const ingestUrl = webhookIngestUrl();
    return renderSectionCard({
      title: 'Bot & webhook',
      desc: 'Configure Telegram Bot API credentials and webhook target.',
      children: [
        el('form', {
          className: 'stack',
          onSubmit: handleSaveBot,
        },
          renderFormField({
            label: 'Bot ID',
            htmlFor: 'tg-bot-id',
            children: el('input', {
              id: 'tg-bot-id',
              className: 'form-input',
              inputMode: 'numeric',
              value: state.bot.bot_id,
              disabled: !canWrite,
              placeholder: 'Telegram bot numeric ID',
              onInput: (e: Event) => { state.bot.bot_id = eventTargetValue(e); },
            }),
          }),
          renderFormField({
            label: state.configured ? `Bot token (saved: ${maskSecret(storedToken)})` : 'Bot token',
            htmlFor: 'tg-bot-token',
            children: el('input', {
              id: 'tg-bot-token',
              className: 'form-input',
              type: 'password',
              autocomplete: 'off',
              disabled: !canWrite,
              placeholder: state.configured ? 'Leave blank to keep current' : 'From @BotFather',
              value: state.tokenInput,
              onInput: (e: Event) => { state.tokenInput = eventTargetValue(e); },
            }),
          }),
          renderFormField({
            label: 'Webhook URL',
            htmlFor: 'tg-webhook-url',
            hint: 'Public URL registered with Telegram via setWebhook',
            children: el('input', {
              id: 'tg-webhook-url',
              className: 'form-input',
              disabled: !canWrite,
              placeholder: 'https://your-domain/api/v1/telegram/webhook/…',
              value: state.bot.webhook_url,
              onInput: (e: Event) => { state.bot.webhook_url = eventTargetValue(e); },
            }),
          }),
          ingestUrl
            ? el('div', { className: 'toolbar-row text-sm' },
              el('span', { className: 'text-muted' }, 'BidShard ingest URL:'),
              el('code', { className: 'code-inline flex-1' }, ingestUrl),
              renderButton({
                label: 'Copy',
                variant: 'secondary',
                size: 'sm',
                onClick: () => copyText(ingestUrl),
              }),
            )
            : null,
          renderFormField({
            label: state.configured ? `Secret token (saved: ${maskSecret(storedSecret)})` : 'Secret token',
            htmlFor: 'tg-secret-token',
            children: el('div', { className: 'flex items-center gap-2' },
              el('input', {
                id: 'tg-secret-token',
                className: 'form-input flex-1',
                type: 'password',
                autocomplete: 'off',
                disabled: !canWrite,
                placeholder: state.configured ? 'Leave blank to keep current' : 'X-Telegram-Bot-Api-Secret-Token',
                value: state.secretInput,
                onInput: (e: Event) => { state.secretInput = eventTargetValue(e); },
              }),
              canWrite
                ? renderButton({
                  label: 'Generate',
                  variant: 'secondary',
                  size: 'sm',
                  onClick: () => {
                    state.secretInput = crypto.randomUUID().replace(/-/g, '');
                    render();
                  },
                })
                : null,
            ),
          }),
          renderFormField({
            label: 'Mini App URL',
            htmlFor: 'tg-mini-app-url',
            children: el('input', {
              id: 'tg-mini-app-url',
              className: 'form-input',
              disabled: !canWrite,
              placeholder: 'https://t.me/your_bot/your_app',
              value: state.bot.mini_app_url,
              onInput: (e: Event) => { state.bot.mini_app_url = eventTargetValue(e); },
            }),
          }),
          renderFormField({
            label: 'Session validation window',
            htmlFor: 'tg-auth-ttl',
            hint: 'How long Telegram login data stays valid, in seconds',
            children: el('input', {
              id: 'tg-auth-ttl',
              className: 'form-input',
              inputMode: 'numeric',
              disabled: !canWrite,
              value: state.bot.auth_date_ttl,
              onInput: (e: Event) => { state.bot.auth_date_ttl = eventTargetValue(e); },
            }),
          }),
          canWrite
            ? renderButton({
              label: state.saving ? 'Saving…' : 'Save bot config',
              variant: 'primary',
              type: 'submit',
              loading: state.saving,
              disabled: state.saving,
            })
            : null,
        ),
      ],
    });
  }

  function renderDeeplinkSection() {
    return renderSectionCard({
      title: 'Deeplink / bridge token',
      desc: 'Create trackable links for Telegram click attribution.',
      children: [
        el('div', { className: 'stack' },
        renderFormField({
          label: 'UTM source (optional)',
          htmlFor: 'tg-utm-source',
          children: el('input', {
            id: 'tg-utm-source',
            className: 'form-input',
            disabled: !canWrite,
            value: state.utmSource,
            onInput: (e: Event) => { state.utmSource = eventTargetValue(e); },
          }),
        }),
        renderFormField({
          label: 'UTM campaign (optional)',
          htmlFor: 'tg-utm-campaign',
          children: el('input', {
            id: 'tg-utm-campaign',
            className: 'form-input',
            disabled: !canWrite,
            value: state.utmCampaign,
            onInput: (e: Event) => { state.utmCampaign = eventTargetValue(e); },
          }),
        }),
        canWrite
          ? renderButton({
            label: 'Generate bridge token',
            variant: 'secondary',
            size: 'sm',
            onClick: handleCreateDeeplink,
          })
          : null,
        state.deeplink?.token
          ? el('div', { className: 'stack stack--sm text-sm' },
            el('p', null, el('strong', null, 'Token: '), el('code', { className: 'font-mono' }, state.deeplink.token)),
            el('p', { className: 'text-muted' },
              'Expires: ',
              state.deeplink.expires_at
                ? new Date(state.deeplink.expires_at).toLocaleString()
                : '—',
            ),
            el('div', { className: 'toolbar-row' },
              el('code', { className: 'code-inline flex-1' }, clickBridgeUrl(state.deeplink.token)),
              renderButton({
                label: 'Copy click URL',
                variant: 'secondary',
                size: 'sm',
                onClick: () => {
                  const token = state.deeplink?.token;
                  if (token) copyText(clickBridgeUrl(token));
                },
              }),
            ),
          )
          : null,
        ),
      ],
    });
  }

  function renderPostbacksSection() {
    const rows = state.postbacks;
    return renderSectionCard({
      title: 'S2S postbacks',
      desc: 'Advertiser conversion callbacks (supports {click_id}, {campaign_id} macros).',
      children: [
        el('div', { className: 'stack' },
          rows.length > 0
          ? el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', null, 'URL'),
                el('th', null, 'Actions'),
              ),
            ),
            el('tbody', null,
              rows.map((row: any) => el('tr', null,
                el('td', null, el('code', { className: 'code-inline' }, row.postback_url)),
                el('td', null,
                  canWrite
                    ? el('div', { className: 'cluster--actions' },
                      renderButton({
                        label: 'Test',
                        variant: 'secondary',
                        size: 'sm',
                        onClick: () => handleTestPostback(row.id),
                      }),
                      renderButton({
                        label: 'Delete',
                        variant: 'danger',
                        size: 'sm',
                        onClick: () => handleDeletePostback(row.id),
                      }),
                    )
                    : null,
                ),
              )),
            ),
          ),
          )
          : renderEmptyState({
            title: 'No postbacks configured',
            description: 'Add an S2S postback URL below.',
            icon: 'send',
          }),
        canWrite
          ? el('form', {
            className: 'stack',
            onSubmit: handleAddPostback,
          },
            renderFormField({
              label: 'Postback URL',
              htmlFor: 'tg-new-postback',
              children: el('input', {
                id: 'tg-new-postback',
                className: 'form-input min-w-80',
                placeholder: 'https://partner.example/postback?cid={click_id}',
                value: state.newPostbackUrl,
                onInput: (e: Event) => { state.newPostbackUrl = eventTargetValue(e); },
              }),
            }),
            renderButton({
              label: state.postbackBusy ? 'Adding…' : 'Add postback',
              variant: 'primary',
              size: 'sm',
              type: 'submit',
              loading: state.postbackBusy,
              disabled: state.postbackBusy,
            }),
          )
          : null,
        ),
      ],
    });
  }

  function render() {
    if (destroyed) return;
    if (state.loading) {
      replaceChildren(container, el('p', { className: 'text-muted' }, 'Loading Telegram settings…'));
      return;
    }
    if (state.error) {
      replaceChildren(container, renderErrorBlock(state.error));
      return;
    }
    replaceChildren(container,
      el('div', { className: 'tg-panel stack stack--lg' },
        el('div', { className: 'page-header page-header--compact' },
          el('h2', { className: 'page-header__title' }, 'Telegram Mini App'),
          el('p', { className: 'page-header__desc text-sm' },
            el('a', { href: `/reports/telegram?campaign_id=${encodeURIComponent(campaignId)}` }, 'Open full analytics'),
            ' · ',
            'Tracking endpoints: ',
            el('code', { className: 'font-mono' }, '/tg/click'),
            ', ',
            el('code', { className: 'font-mono' }, '/tg/impression'),
          ),
        ),
        renderBotSection(),
        renderDeeplinkSection(),
        renderPostbacksSection(),
      ),
    );
  }

  load();

  return {
    destroy() {
      destroyed = true;
      gate.release();
    },
  };
}
