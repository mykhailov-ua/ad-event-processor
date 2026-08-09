import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import {
  fetchPostbackConfig,
  fetchPostbackDlq,
  retryPostbackDlq,
  savePostbackConfig,
} from '../helpers/postback_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';

/**
 * Mount S2S postback config + DLQ panel for a campaign.
 *
 * @param {HTMLElement} container
 * @param {{ campaignId: string, canWrite: boolean }} opts
 * @returns {{ destroy: () => void }}
 */
export function mountCampaignPostbackPanel(container, opts) {
  let destroyed = false;
  const form = {
    provider: 'webhook',
    url_template: '',
    api_token: '',
    target_event: 'conversion',
  };
  let dlq = [];
  let loading = true;
  let saving = false;
  let error = null;

  async function load() {
    loading = true;
    render();
    const [cfg, dlqRes] = await Promise.all([
      fetchPostbackConfig(opts.campaignId),
      fetchPostbackDlq(opts.campaignId),
    ]);
    if (destroyed) return;
    if (cfg) {
      form.provider = cfg.provider || 'webhook';
      form.url_template = cfg.url_template || '';
      form.target_event = cfg.target_event || 'conversion';
    }
    dlq = dlqRes;
    loading = false;
    render();
  }

  async function save() {
    if (!opts.canWrite) return;
    saving = true;
    error = null;
    render();
    const [, err] = await to(savePostbackConfig(opts.campaignId, {
      provider: form.provider,
      url_template: form.url_template,
      api_token: form.api_token || undefined,
      target_event: form.target_event,
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

  async function retry(id) {
    const [, err] = await to(retryPostbackDlq(id));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Retry failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Retry queued', message: `DLQ #${id}` });
    load();
  }

  function render() {
    container.replaceChildren(
      el('div', { className: 'section-card stack' },
        el('h3', { className: 'subsection-title' }, 'S2S postback'),
        loading ? el('p', { className: 'text-muted' }, 'Loading…') : null,
        error ? el('p', { className: 'text-danger text-sm' }, error) : null,
        el('label', { className: 'form-field', htmlFor: 'pb-provider' },
          'Provider',
          el('select', {
            id: 'pb-provider',
            className: 'form-input form-input--sm',
            disabled: !opts.canWrite,
            onChange: (e) => { form.provider = e.target.value; },
          },
            el('option', { value: 'webhook', selected: form.provider === 'webhook' }, 'Webhook'),
            el('option', { value: 'facebook', selected: form.provider === 'facebook' }, 'Facebook'),
          ),
        ),
        el('label', { className: 'form-field', htmlFor: 'pb-url' },
          'URL template',
          el('input', {
            id: 'pb-url',
            className: 'form-input',
            disabled: !opts.canWrite,
            placeholder: 'https://partner.example/postback?cid={click_id}',
            value: form.url_template,
            onInput: (e) => { form.url_template = e.target.value; },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'pb-token' },
          'API token (optional, masked after save)',
          el('input', {
            id: 'pb-token',
            type: 'password',
            className: 'form-input',
            disabled: !opts.canWrite,
            value: form.api_token,
            onInput: (e) => { form.api_token = e.target.value; },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'pb-event' },
          'Target event',
          el('select', {
            id: 'pb-event',
            className: 'form-input form-input--sm',
            disabled: !opts.canWrite,
            onChange: (e) => { form.target_event = e.target.value; },
          },
            el('option', { value: 'conversion', selected: form.target_event === 'conversion' }, 'Conversion'),
            el('option', { value: 'click', selected: form.target_event === 'click' }, 'Click'),
            el('option', { value: 'install', selected: form.target_event === 'install' }, 'Install'),
          ),
        ),
        opts.canWrite
          ? el('button', {
            type: 'button',
            className: 'btn btn--primary btn--sm',
            disabled: saving,
            onClick: save,
          }, saving ? 'Saving…' : 'Save postback')
          : null,
        el('h4', { className: 'subsection-title mt-4' }, 'DLQ'),
        dlq.length === 0
          ? el('p', { className: 'text-muted text-sm' }, 'No failed postbacks.')
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
                dlq.map((row) => el('tr', null,
                  el('td', null, String(row.id)),
                  el('td', null, row.event_type ?? '—'),
                  el('td', null, String(row.failures_count ?? 0)),
                  el('td', null, row.status ?? '—'),
                  el('td', null,
                    opts.canWrite && row.status !== 'RETRIED'
                      ? el('button', {
                        type: 'button',
                        className: 'btn btn--secondary btn--sm',
                        onClick: () => retry(row.id),
                      }, 'Retry')
                      : null,
                  ),
                )),
              ),
            ),
          ),
      ),
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
