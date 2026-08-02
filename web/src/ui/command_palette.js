import { el } from '../lib/dom.js';
import { navigate } from '../lib/router.js';
import { renderIcon } from './icon.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import { NAV_GROUPS, navLinkVisible } from '../helpers/nav_config.js';
import { isCustomerUuid, shortCustomerId } from '../helpers/customer_context.js';

/**
 * @returns {{ destroy: () => void }}
 */
export function installCommandPalette() {
  let overlay = null;
  let input = null;
  let listEl = null;
  let highlight = 0;
  let filtered = [];

  function destroy() {
    if (overlay) overlay.remove();
    overlay = null;
    document.removeEventListener('keydown', onGlobalKey);
  }

  function buildItems() {
    const user = auth.getUser();
    const permissions = user?.permissions ?? [];
    /** @type {{ id: string, label: string, hint?: string, run: () => void }[]} */
    const items = [];

    for (const group of NAV_GROUPS) {
      for (const link of group.links) {
        if (!navLinkVisible(permissions, link)) continue;
        items.push({
          id: `nav-${link.to}`,
          label: link.label,
          hint: group.title,
          run: () => {
            close();
            navigate(link.to);
          },
        });
      }
    }

    for (const id of storage.getRecentCustomerIds()) {
      items.push({
        id: `customer-${id}`,
        label: shortCustomerId(id, 12),
        hint: 'Recent customer',
        run: () => {
          close();
          navigate(`/customers/${id}`);
        },
      });
      items.push({
        id: `billing-${id}`,
        label: `Billing · ${shortCustomerId(id, 12)}`,
        hint: 'Customer billing',
        run: () => {
          close();
          navigate(`/billing?customer_id=${encodeURIComponent(id)}`);
        },
      });
      items.push({
        id: `campaigns-${id}`,
        label: `Campaigns · ${shortCustomerId(id, 12)}`,
        hint: 'Customer campaigns',
        run: () => {
          close();
          navigate(`/campaigns?customer_id=${encodeURIComponent(id)}`);
        },
      });
    }

    return items;
  }

  function filterItems(query) {
    const q = query.trim().toLowerCase();
    const all = buildItems();
    if (!q) return all.slice(0, 12);
    const matches = all.filter((item) =>
      item.label.toLowerCase().includes(q)
      || (item.hint?.toLowerCase().includes(q))
      || item.id.toLowerCase().includes(q),
    );
    if (isCustomerUuid(q)) {
      const id = query.trim();
      matches.unshift({
        id: `goto-customer-${id}`,
        label: `Open customer ${shortCustomerId(id, 12)}`,
        hint: 'UUID match',
        run: () => {
          close();
          navigate(`/customers/${id}`);
        },
      });
    }
    return matches.slice(0, 12);
  }

  function renderList() {
    if (!listEl) return;
    listEl.replaceChildren();
    filtered.forEach((item, i) => {
      const row = el('button', {
        type: 'button',
        className: 'cmd-palette__item' + (i === highlight ? ' cmd-palette__item--active' : ''),
        onMouseEnter: () => {
          highlight = i;
          renderList();
        },
        onClick: () => item.run(),
      },
        el('span', { className: 'cmd-palette__item-label' }, item.label),
        item.hint
          ? el('span', { className: 'cmd-palette__item-hint' }, item.hint)
          : null,
      );
      listEl.appendChild(row);
    });
    if (filtered.length === 0) {
      listEl.appendChild(el('div', { className: 'cmd-palette__empty' }, 'No matches'));
    }
  }

  function open() {
    if (overlay) return;
    highlight = 0;
    filtered = filterItems('');

    overlay = el('div', {
      className: 'cmd-palette-overlay',
      role: 'presentation',
      onClick: () => close(),
    });

    const dialog = el('div', {
      className: 'cmd-palette',
      role: 'dialog',
      'aria-modal': 'true',
      'aria-label': 'Command palette',
      onClick: (e) => e.stopPropagation(),
    });

    const inputWrap = el('div', { className: 'cmd-palette__input-wrap' },
      renderIcon('search', { size: 18 }),
    );

    input = el('input', {
      type: 'search',
      className: 'cmd-palette__input',
      placeholder: 'Jump to page, customer, billing…',
      autocomplete: 'off',
      onInput: (e) => {
        filtered = filterItems(e.target.value);
        highlight = 0;
        renderList();
      },
      onKeydown: (e) => {
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          highlight = Math.min(highlight + 1, filtered.length - 1);
          renderList();
        } else if (e.key === 'ArrowUp') {
          e.preventDefault();
          highlight = Math.max(highlight - 1, 0);
          renderList();
        } else if (e.key === 'Enter') {
          e.preventDefault();
          if (filtered[highlight]) filtered[highlight].run();
        } else if (e.key === 'Escape') {
          e.preventDefault();
          close();
        }
      },
    });

    inputWrap.appendChild(input);

    listEl = el('div', { className: 'cmd-palette__list' });
    renderList();

    dialog.appendChild(inputWrap);
    dialog.appendChild(listEl);
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
    input.focus();
  }

  function close() {
    destroy();
  }

  function onGlobalKey(e) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      if (overlay) close();
      else open();
      return;
    }
    if (e.key === 'Escape' && overlay) {
      e.preventDefault();
      close();
    }
  }

  document.addEventListener('keydown', onGlobalKey);

  return { destroy: () => {
    close();
    document.removeEventListener('keydown', onGlobalKey);
  } };
}
