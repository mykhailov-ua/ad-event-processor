import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren } from '../lib/dom.js';
import * as auth from '../helpers/auth.js';
import { can, maskLevel } from '../helpers/permissions.js';
import { mountCampaignTelegramPanel } from './campaign_telegram_panel.js';

/**
 * Standalone Telegram config page for a campaign (`/campaigns/:id/telegram`).
 *
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string> }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const campaignId = ctx.params.id;
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const masked = maskLevel(permissions) === 'masked';
  const canWrite = can(permissions, 'campaigns:write');

  const slot = el('div', { 'data-tg-panel': '' });
  /** @type {{ destroy: () => void }|null} */
  let panelHandle: any = null;

  function render() {
    if (destroyed) return;
    if (masked) {
      replaceChildren(container,
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Telegram'),
        ),
        el('p', null, 'Telegram configuration is not available for masked accounts.'),
        el('a', { href: `/campaigns/${encodeURIComponent(campaignId)}` }, 'Back to campaign'),
      );
      return;
    }

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, 'Telegram Mini App'),
        el('p', { className: 'text-muted' },
          el('a', { href: `/campaigns/${encodeURIComponent(campaignId)}` }, '← Campaign'),
          ' · ',
          el('a', { href: `/reports/telegram?campaign_id=${encodeURIComponent(campaignId)}` }, 'Open full analytics'),
        ),
      ),
      slot,
    );

    if (!panelHandle) {
      panelHandle = mountCampaignTelegramPanel(slot, { campaignId, canWrite });
    }
  }

  render();

  return {
    destroy() {
      destroyed = true;
      panelHandle?.destroy();
      panelHandle = null;
    },
  };
}
