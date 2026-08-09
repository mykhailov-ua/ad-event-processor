import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { buildTrackingLink, defaultClickTemplate } from '../helpers/tracking_link.js';
import { pushToastMessage } from '../helpers/toast_ui.js';

/**
 * Mount per-campaign tracking link generator panel.
 *
 * @param {HTMLElement} container
 * @param {{ campaignId: string }} opts
 * @returns {{ destroy: () => void }}
 */
export function mountCampaignTrackingPanel(container, opts) {
  let destroyed = false;
  const subs = { sub1: '', sub2: '', sub3: '', sub4: '', sub5: '' };
  let template = defaultClickTemplate('');

  async function loadTemplate() {
    const [docRes, platRes] = await Promise.all([
      to(api('/api/v1/ops/doctor')),
      to(api('/api/v1/settings/platform')),
    ]);
    if (destroyed) return;
    template = docRes[0]?.data?.click_url_template
      || defaultClickTemplate(platRes[0]?.data?.config?.tracking_domain ?? '');
    render();
  }

  function copyLink() {
    const link = buildTrackingLink(template, opts.campaignId, subs);
    navigator.clipboard?.writeText(link).then(() => {
      pushToastMessage({ title: 'Copied', message: 'Tracking link copied to clipboard' });
    }).catch(() => {
      pushToastMessage({ title: 'Copy failed', message: link });
    });
  }

  function render() {
    const link = buildTrackingLink(template, opts.campaignId, subs);
    container.replaceChildren(
      el('div', { className: 'section-card stack' },
        el('h3', { className: 'subsection-title' }, 'Tracking link'),
        el('p', { className: 'text-muted text-sm' },
          'Substitute sub parameters for source tracking. Template comes from platform tracking domain.',
        ),
        ...[1, 2, 3, 4, 5].map((n) => {
          const key = `sub${n}`;
          return el('label', { className: 'form-field', htmlFor: `track-${key}` },
            key.toUpperCase(),
            el('input', {
              id: `track-${key}`,
              className: 'form-input form-input--sm',
              placeholder: `Value for ${key}`,
              value: subs[key],
              onInput: (e) => {
                subs[key] = e.target.value;
                render();
              },
            }),
          );
        }),
        el('div', { className: 'form-field' },
          el('span', { className: 'form-label' }, 'Generated URL'),
          el('code', { className: 'code-block' }, link),
        ),
        el('button', {
          type: 'button',
          className: 'btn btn--secondary btn--sm',
          onClick: copyLink,
        }, 'Copy link'),
      ),
    );
  }

  render();
  loadTemplate();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
