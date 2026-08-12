import type { ViewHandle } from '../lib/router_types.js';
import { el, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import {
  fetchPostbackConfig,
  fetchPostbackDlq,
  retryPostbackDlq,
  savePostbackConfig,
  type PostbackConfigRow,
} from '../helpers/postback_api.js';
import {
  normalizePostbackProvider,
  postbackProviderIds,
  POSTBACK_PROVIDER_UI,
  type PostbackProvider,
} from '../helpers/postback_provider_ui.js';
import {
  AFFILIATE_POSTBACK_PRESETS,
  affiliatePostbackById,
} from '../models/affiliate_postback_presets.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { renderButton } from '../ui/button.js';
import { renderEmptyState } from '../ui/data_table.js';

export type CampaignPostbackPanelOpts = {
  campaignId: string;
  canWrite: boolean;
  navigate?: (path: string) => void;
};

type PostbackForm = {
  provider: PostbackProvider;
  url_template: string;
  api_token: string;
  target_event: string;
  test_event_code: string;
  affiliate_preset_id: string;
};

/**
 * Copy text to the clipboard and show a toast.
 */
function copyText(label: string, text: string): void {
  navigator.clipboard?.writeText(text).then(() => {
    pushToastMessage({ title: 'Copied', message: `${label} copied to clipboard` });
  }).catch(() => {
    pushToastMessage({ title: 'Copy failed', message: text || '(empty)' });
  });
}

/**
 * Mount S2S postback / CAPI config + DLQ panel for a campaign.
 */
export function mountCampaignPostbackPanel(
  container: HTMLElement,
  opts: CampaignPostbackPanelOpts,
): ViewHandle {
  let destroyed = false;
  const form: PostbackForm = {
    provider: 'webhook',
    url_template: '',
    api_token: '',
    target_event: 'conversion',
    test_event_code: '',
    affiliate_preset_id: '',
  };
  let dlq: PostbackConfigRow[] = [];
  let loading = true;
  let saving = false;
  let error: string | null = null;

  async function load(): Promise<void> {
    loading = true;
    render();
    const [cfg, dlqRes] = await Promise.all([
      fetchPostbackConfig(opts.campaignId),
      fetchPostbackDlq(opts.campaignId),
    ]);
    if (destroyed) return;
    if (cfg) {
      form.provider = normalizePostbackProvider(String(cfg.provider || 'webhook'));
      form.url_template = String(cfg.url_template || '');
      form.target_event = String(cfg.target_event || 'conversion');
      form.test_event_code = String(cfg.test_event_code || '');
    }
    dlq = dlqRes;
    loading = false;
    render();
  }

  async function save(): Promise<void> {
    if (!opts.canWrite) return;
    saving = true;
    error = null;
    render();
    const [, err] = await to(savePostbackConfig(opts.campaignId, {
      provider: form.provider,
      url_template: form.url_template,
      api_token: form.api_token || undefined,
      target_event: form.target_event,
      test_event_code: form.test_event_code || undefined,
    }));
    if (destroyed) return;
    saving = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      error = mapServiceError(err).message;
      render();
      return;
    }
    form.api_token = '';
    pushToastMessage({ title: 'Postback saved', message: 'Configuration updated' });
    load();
  }

  async function retry(id: string | number): Promise<void> {
    const [, err] = await to(retryPostbackDlq(id));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Retry failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Retry queued', message: `DLQ #${id}` });
    load();
  }

  function applyAffiliatePreset(id: string): void {
    form.affiliate_preset_id = id;
    if (!id) {
      render();
      return;
    }
    const preset = affiliatePostbackById(id);
    if (!preset) {
      render();
      return;
    }
    form.provider = 'webhook';
    form.url_template = preset.url_template;
    render();
  }

  function render(): void {
    const ui = POSTBACK_PROVIDER_UI[form.provider];
    const selectedPreset = form.affiliate_preset_id
      ? affiliatePostbackById(form.affiliate_preset_id)
      : null;

    container.replaceChildren(
      el('div', { className: 'section-card stack', 'data-testid': 'campaign-capi-panel' },
        el('h3', { className: 'subsection-title' }, 'CAPI & Postbacks'),
        el('p', { className: 'text-muted text-sm' },
          'When a tracked event matches the target type, the postback worker dispatches to the provider below. ',
          'CAPI adapters use click IDs captured on redirect (/click) or zero-redirect /track (fbclid, gclid, ttclid). ',
          'Inbound affiliate S2S (partner → BidShard) is configured on the ',
          el('a', {
            href: `/campaigns/${opts.campaignId}?tab=tracking`,
            className: 'text-sm',
            onClick: (e: Event) => {
              if (!opts.navigate) return;
              e.preventDefault();
              opts.navigate(`/campaigns/${opts.campaignId}?tab=tracking`);
            },
          }, 'Integration'),
          ' tab.',
        ),
        loading ? el('p', { className: 'text-muted' }, 'Loading…') : null,
        error ? el('p', { className: 'text-danger text-sm' }, error) : null,

        el('label', { className: 'form-field', htmlFor: 'pb-provider' },
          'Provider',
          el('select', {
            id: 'pb-provider',
            className: 'form-input form-input--sm',
            disabled: !opts.canWrite,
            onChange: (e: Event) => {
              form.provider = normalizePostbackProvider(eventTargetValue(e));
              render();
            },
          },
            ...postbackProviderIds().map((id) => el('option', {
              value: id,
              selected: form.provider === id,
            }, POSTBACK_PROVIDER_UI[id].label)),
          ),
        ),
        el('p', { className: 'text-muted text-sm' }, ui.blurb),

        form.provider === 'webhook'
          ? el('label', {
            className: 'form-field',
            htmlFor: 'pb-affiliate-preset',
            'data-testid': 'affiliate-postback-preset-field',
          },
            'Affiliate network preset',
            el('select', {
              id: 'pb-affiliate-preset',
              className: 'form-input form-input--sm',
              'data-testid': 'affiliate-postback-preset',
              disabled: !opts.canWrite,
              onChange: (e: Event) => applyAffiliatePreset(eventTargetValue(e)),
            },
              el('option', { value: '', selected: !form.affiliate_preset_id }, '— Select network —'),
              ...AFFILIATE_POSTBACK_PRESETS.map((p) => el('option', {
                value: p.id,
                selected: form.affiliate_preset_id === p.id,
              }, p.name)),
            ),
          )
          : null,

        selectedPreset
          ? el('p', { className: 'text-muted text-sm', 'data-testid': 'affiliate-postback-notes' },
            selectedPreset.notes
              || `Network tokens: click=${selectedPreset.network_click_token}, payout=${selectedPreset.network_payout_token}. `
                + 'Replace REPLACE_HOST with the partner postback host. Outbound macros are BidShard {click_id}/{payout}/{tx_id}.',
          )
          : null,

        el('label', { className: 'form-field', htmlFor: 'pb-url' },
          ui.primaryLabel,
          el('div', { className: 'toolbar-row' },
            el('input', {
              id: 'pb-url',
              className: 'form-input',
              'data-testid': 'postback-url-template',
              disabled: !opts.canWrite,
              placeholder: ui.primaryPlaceholder,
              value: form.url_template,
              onInput: (e: Event) => { form.url_template = eventTargetValue(e); },
            }),
            renderButton({
              label: 'Copy',
              variant: 'secondary',
              size: 'sm',
              testId: 'postback-url-copy',
              disabled: !form.url_template,
              onClick: () => copyText(ui.primaryLabel, form.url_template),
            }),
          ),
          el('span', { className: 'form-hint text-muted text-sm' }, ui.primaryHelp),
        ),

        el('label', { className: 'form-field', htmlFor: 'pb-token' },
          ui.tokenLabel,
          el('input', {
            id: 'pb-token',
            type: 'password',
            className: 'form-input',
            disabled: !opts.canWrite,
            placeholder: ui.tokenPlaceholder,
            value: form.api_token,
            onInput: (e: Event) => { form.api_token = eventTargetValue(e); },
          }),
          el('span', { className: 'form-hint text-muted text-sm' }, ui.tokenHelp),
        ),

        el('label', { className: 'form-field', htmlFor: 'pb-event' },
          'Target event',
          el('select', {
            id: 'pb-event',
            className: 'form-input form-input--sm',
            disabled: !opts.canWrite,
            onChange: (e: Event) => { form.target_event = eventTargetValue(e); },
          },
            el('option', { value: 'conversion', selected: form.target_event === 'conversion' }, 'Conversion'),
            el('option', { value: 'click', selected: form.target_event === 'click' }, 'Click'),
            el('option', { value: 'install', selected: form.target_event === 'install' }, 'Install'),
            el('option', { value: 'lead', selected: form.target_event === 'lead' }, 'Lead'),
            el('option', { value: 'purchase', selected: form.target_event === 'purchase' }, 'Purchase'),
          ),
          el('span', { className: 'form-hint text-muted text-sm' }, ui.eventMappingHint),
        ),

        ui.supportsTestEventCode
          ? el('label', {
            className: 'form-field',
            htmlFor: 'pb-test-event-code',
            'data-testid': 'postback-test-event-code-field',
          },
            'Test event code (Meta / TikTok staging)',
            el('input', {
              id: 'pb-test-event-code',
              className: 'form-input form-input--sm',
              'data-testid': 'postback-test-event-code',
              disabled: !opts.canWrite,
              placeholder: 'TEST12345',
              value: form.test_event_code,
              onInput: (e: Event) => { form.test_event_code = eventTargetValue(e); },
            }),
            el('span', { className: 'form-hint text-muted text-sm' },
              'Routes events to the provider test stream. Use with scripts/test/capi_meta_staging.sh on staging.',
            ),
          )
          : null,

        opts.canWrite
          ? renderButton({
            label: saving ? 'Saving…' : 'Save postback',
            variant: 'primary',
            size: 'sm',
            loading: saving,
            disabled: saving,
            onClick: save,
          })
          : null,

        el('h4', { className: 'subsection-title mt-4' }, 'DLQ'),
        dlq.length === 0
          ? renderEmptyState({
            title: 'No failed postbacks',
            description: 'DLQ entries appear when outbound postbacks fail.',
            icon: 'send',
          })
          : el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'ID'),
                  el('th', { scope: 'col' }, 'Event'),
                  el('th', { scope: 'col' }, 'Failures'),
                  el('th', { scope: 'col' }, 'Status'),
                  el('th', { scope: 'col' }, ''),
                ),
              ),
              el('tbody', null,
                dlq.map((row) => {
                  const rowId = row.id;
                  const status = typeof row.status === 'string' ? row.status : '';
                  return el('tr', null,
                    el('td', null, String(rowId ?? '')),
                    el('td', null, typeof row.event_type === 'string' ? row.event_type : '—'),
                    el('td', null, String(row.failures_count ?? 0)),
                    el('td', null, status || '—'),
                    el('td', null,
                      opts.canWrite && status !== 'RETRIED' && (typeof rowId === 'string' || typeof rowId === 'number')
                        ? renderButton({
                          label: 'Retry',
                          variant: 'secondary',
                          size: 'sm',
                          onClick: () => retry(rowId),
                        })
                        : null,
                    ),
                  );
                }),
              ),
            ),
          ),
      ),
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
